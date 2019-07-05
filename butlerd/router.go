package butlerd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/itchio/wharf/werrors"
	"golang.org/x/sync/singleflight"

	"github.com/itchio/headway/tracker"
	"github.com/itchio/httpkit/neterr"
	"github.com/itchio/httpkit/timeout"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd/horror"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/headway/state"
	"github.com/pkg/errors"
	"github.com/sourcegraph/jsonrpc2"
)

type InFlightRequest struct {
	DispatchedAt time.Time
	Desc         string
}

type BackgroundTaskID int64

type InFlightBackgroundTask struct {
	QueuedAt time.Time
	Desc     string
}

type BackgroundTask struct {
	Desc string
	Do   func(rc *RequestContext) error
}

type RequestHandler func(rc *RequestContext) (interface{}, error)
type NotificationHandler func(rc *RequestContext)

type GetClientFunc func(key string) *itchio.Client

type Router struct {
	Handlers             map[string]RequestHandler
	NotificationHandlers map[string]NotificationHandler
	CancelFuncs          *CancelFuncs
	dbPool               *sqlite.Pool
	getClient            GetClientFunc
	httpClient           *http.Client
	httpTransport        *http.Transport

	Group                *singleflight.Group
	ShutdownChan         chan struct{}
	initiateShutdownOnce sync.Once
	completeShutdownOnce sync.Once
	shuttingDown         bool
	backgroundContext    context.Context
	backgroundCancel     context.CancelFunc

	inflightRequests        map[jsonrpc2.ID]InFlightRequest
	inflightBackgroundTasks map[BackgroundTaskID]InFlightBackgroundTask
	inflightLock            sync.Mutex

	backgroundTaskIDSeed BackgroundTaskID

	ButlerVersion       string
	ButlerVersionString string

	globalConsumer *state.Consumer
}

func NewRouter(dbPool *sqlite.Pool, getClient GetClientFunc, httpClient *http.Client, httpTransport *http.Transport) *Router {
	backgroundContext, backgroundCancel := context.WithCancel(context.Background())

	return &Router{
		Handlers:             make(map[string]RequestHandler),
		NotificationHandlers: make(map[string]NotificationHandler),
		CancelFuncs: &CancelFuncs{
			Funcs: make(map[string]context.CancelFunc),
		},
		dbPool:        dbPool,
		getClient:     getClient,
		httpClient:    httpClient,
		httpTransport: httpTransport,

		backgroundContext: backgroundContext,
		backgroundCancel:  backgroundCancel,

		inflightRequests:        make(map[jsonrpc2.ID]InFlightRequest),
		inflightBackgroundTasks: make(map[BackgroundTaskID]InFlightBackgroundTask),

		Group:        &singleflight.Group{},
		ShutdownChan: make(chan struct{}),

		backgroundTaskIDSeed: 0,

		globalConsumer: &state.Consumer{
			OnMessage: func(lvl string, msg string) {
				comm.Logf("[router] [%s] %s", lvl, msg)
			},
		},
	}
}

func (r *Router) Register(method string, rh RequestHandler) {
	if _, ok := r.Handlers[method]; ok {
		panic(fmt.Sprintf("Can't register handler twice for %s", method))
	}
	r.Handlers[method] = rh
}

func (r *Router) initiateShutdown() {
	r.initiateShutdownOnce.Do(func() {
		r.Logf("Initiating graceful butlerd shutdown")
		r.inflightLock.Lock()
		r.shuttingDown = true
		if r.numInflightItems() == 0 {
			r.completeShutdown()
		}
		r.inflightLock.Unlock()
		r.backgroundCancel()
	})
}

func (r *Router) numInflightItems() int {
	return len(r.inflightRequests) + len(r.inflightBackgroundTasks)
}

// caller must hold inflightLock
func (r *Router) onRequestStarted(id jsonrpc2.ID, req InFlightRequest) {
	r.inflightRequests[id] = req
}

// caller must hold inflightLock
func (r *Router) onRequestFinished(id jsonrpc2.ID) {
	delete(r.inflightRequests, id)
	if r.shuttingDown {
		r.globalConsumer.Infof("While shutting down, request %s has completed", id)
	}
	r.opportunisticShutdown()
}

// caller must hold inflightLock
func (r *Router) generateBackgroundTaskID() BackgroundTaskID {
	id := r.backgroundTaskIDSeed
	r.backgroundTaskIDSeed += 1
	return id
}

// caller must hold inflightLock
func (r *Router) onBackgroundTaskQueued(id BackgroundTaskID, task InFlightBackgroundTask) {
	r.inflightBackgroundTasks[id] = task
}

// caller must hold inflightLock
func (r *Router) onBackgroundTaskFinished(id BackgroundTaskID) {
	delete(r.inflightBackgroundTasks, id)
	if r.shuttingDown {
		r.globalConsumer.Infof("While shutting down, task %d has completed", id)
	}
	r.opportunisticShutdown()
}

// caller must hold inflightLock
func (r *Router) opportunisticShutdown() {
	if !r.shuttingDown {
		return
	}

	numInFlightItems := r.numInflightItems()
	if numInFlightItems == 0 {
		r.completeShutdown()
	} else {
		r.globalConsumer.Infof("In-flight requests/background tasks preventing shutdown: ")
		for _, req := range r.inflightRequests {
			r.Logf(" - %s (%v)", req.Desc, time.Since(req.DispatchedAt))
		}
		for _, task := range r.inflightBackgroundTasks {
			r.Logf(" - %s (%v)", task.Desc, time.Since(task.QueuedAt))
		}
	}
}

func (r *Router) completeShutdown() {
	r.completeShutdownOnce.Do(func() {
		r.Logf("No in-flight requests left, we can shut down now.")
		close(r.ShutdownChan)
	})
}

func (r *Router) RegisterNotification(method string, nh NotificationHandler) {
	if _, ok := r.NotificationHandlers[method]; ok {
		panic(fmt.Sprintf("Can't register handler twice for %s", method))
	}
	r.NotificationHandlers[method] = nh
}

func (r *Router) Dispatch(ctx context.Context, origConn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	r.inflightLock.Lock()
	r.onRequestStarted(req.ID, InFlightRequest{
		DispatchedAt: time.Now().UTC(),
		Desc:         fmt.Sprintf("[req %v] %s", req.ID, req.Method),
	})
	r.inflightLock.Unlock()

	defer func() {
		r.inflightLock.Lock()
		r.onRequestFinished(req.ID)
		r.inflightLock.Unlock()
	}()

	method := req.Method
	var res interface{}

	conn := &JsonRPC2Conn{origConn}
	consumer, cErr := NewStateConsumer(&NewStateConsumerParams{
		Ctx:  ctx,
		Conn: conn,
	})
	if cErr != nil {
		return
	}

	err := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				if rErr, ok := r.(error); ok {
					err = errors.WithStack(rErr)
				} else {
					err = errors.Errorf("panic: %v", r)
				}
			}
		}()

		rc := &RequestContext{
			Ctx:         ctx,
			Consumer:    consumer,
			Params:      req.Params,
			Conn:        conn,
			CancelFuncs: r.CancelFuncs,
			dbPool:      r.dbPool,
			Client:      r.getClient,

			HTTPClient:    r.httpClient,
			HTTPTransport: r.httpTransport,

			ButlerVersion:       r.ButlerVersion,
			ButlerVersionString: r.ButlerVersionString,

			Group:    r.Group,
			Shutdown: r.initiateShutdown,

			origConn: origConn,
			method:   method,

			QueueBackgroundTask: r.QueueBackgroundTask,
		}

		if req.Notif {
			if nh, ok := r.NotificationHandlers[req.Method]; ok {
				nh(rc)
			}
		} else {
			if h, ok := r.Handlers[method]; ok {
				rc.Consumer.OnProgress = func(alpha float64) {
					if rc.tracker == nil {
						// skip
						return
					}

					rc.tracker.SetProgress(alpha)
					notif := ProgressNotification{
						Progress: alpha,
					}
					stats := rc.tracker.Stats()
					if stats != nil {
						if stats.TimeLeft() != nil {
							notif.ETA = stats.TimeLeft().Seconds()
						}
						if stats.BPS() != nil {
							notif.BPS = stats.BPS().Value
						} else {
							notif.BPS = timeout.GetBPS()
						}
					}
					// cannot use autogenerated wrappers to avoid import cycles
					rc.Notify("Progress", notif)
				}
				rc.Consumer.OnProgressLabel = func(label string) {
					// muffin
				}
				rc.Consumer.OnPauseProgress = func() {
					if rc.tracker != nil {
						rc.tracker.Pause()
					}
				}
				rc.Consumer.OnResumeProgress = func() {
					if rc.tracker != nil {
						rc.tracker.Resume()
					}
				}

				res, err = h(rc)
			} else {
				err = &RpcError{
					Code:    jsonrpc2.CodeMethodNotFound,
					Message: fmt.Sprintf("Method '%s' not found", req.Method),
				}
			}
		}
		return
	}()

	if req.Notif {
		return
	}

	if err == nil {
		err = origConn.Reply(ctx, req.ID, res)
		if err != nil {
			consumer.Errorf("Error while replying: %s", err.Error())
		}
		return
	}

	var code int64
	var message string
	var data map[string]interface{}

	if ee, ok := AsButlerdError(err); ok {
		code = ee.RpcErrorCode()
		message = ee.RpcErrorMessage()
		data = ee.RpcErrorData()
	} else {
		if neterr.IsNetworkError(err) {
			code = int64(CodeNetworkDisconnected)
			message = CodeNetworkDisconnected.Error()
		} else if errors.Cause(err) == werrors.ErrCancelled {
			code = int64(CodeOperationCancelled)
			message = CodeOperationCancelled.Error()
		} else {
			code = jsonrpc2.CodeInternalError
			message = err.Error()
		}
	}

	var rawData *json.RawMessage
	if data == nil {
		data = make(map[string]interface{})
	}
	data["stack"] = fmt.Sprintf("%+v", err)
	data["butlerVersion"] = r.ButlerVersionString

	if ae, ok := itchio.AsAPIError(err); ok {
		code = int64(CodeAPIError)
		data["apiError"] = ae
	}

	marshalledData, marshalErr := json.Marshal(data)
	if marshalErr == nil {
		rawMessage := json.RawMessage(marshalledData)
		rawData = &rawMessage
	}

	origConn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
		Code:    code,
		Message: message,
		Data:    rawData,
	})
}

func (r *Router) doBackgroundTask(id BackgroundTaskID, bt BackgroundTask) {
	defer func() {
		router := r
		if r := recover(); r != nil {
			router.Logf("background task panicked: %+v", r)
		}
	}()

	defer func() {
		r.inflightLock.Lock()
		r.onBackgroundTaskFinished(id)
		r.inflightLock.Unlock()
	}()

	consumer := r.globalConsumer
	rc := &RequestContext{
		Ctx:         r.backgroundContext,
		Consumer:    consumer,
		Params:      nil,
		Conn:        nil,
		CancelFuncs: r.CancelFuncs,
		dbPool:      r.dbPool,
		Client:      r.getClient,

		HTTPClient:    r.httpClient,
		HTTPTransport: r.httpTransport,

		ButlerVersion:       r.ButlerVersion,
		ButlerVersionString: r.ButlerVersionString,

		Group:    r.Group,
		Shutdown: r.initiateShutdown,

		origConn: nil,
		method:   "",

		QueueBackgroundTask: r.QueueBackgroundTask,
	}

	err := func() (retErr error) {
		defer horror.RecoverInto(&retErr)
		consumer.Debugf("Executing background task %d: %s", id, bt.Desc)
		return bt.Do(rc)
	}()
	if err != nil {
		consumer.Warnf("Background task error: %+v", err)
	}
}

func (r *Router) QueueBackgroundTask(bt BackgroundTask) {
	r.inflightLock.Lock()
	id := r.generateBackgroundTaskID()
	r.onBackgroundTaskQueued(id, InFlightBackgroundTask{
		QueuedAt: time.Now().UTC(),
		Desc:     fmt.Sprintf("[task %d] %s", id, bt.Desc),
	})
	r.inflightLock.Unlock()

	go r.doBackgroundTask(id, bt)
}

func (r *Router) Logf(format string, args ...interface{}) {
	r.globalConsumer.Infof(format, args...)
}

type BackgroundTaskFunc func(rc *RequestContext) error

type RequestContext struct {
	Ctx                 context.Context
	Consumer            *state.Consumer
	Client              GetClientFunc
	QueueBackgroundTask func(bt BackgroundTask)

	HTTPClient    *http.Client
	HTTPTransport *http.Transport

	Params      *json.RawMessage
	Conn        Conn
	CancelFuncs *CancelFuncs
	dbPool      *sqlite.Pool

	ButlerVersion       string
	ButlerVersionString string

	Group    *singleflight.Group
	Shutdown func()

	notificationInterceptors map[string]NotificationInterceptor
	tracker                  tracker.Tracker

	method   string
	origConn *jsonrpc2.Conn
}

type WithParamsFunc func() (interface{}, error)

type NotificationInterceptor func(method string, params interface{}) error

func (rc *RequestContext) Call(method string, params interface{}, res interface{}) error {
	return rc.Conn.Call(rc.Ctx, method, params, res)
}

func (rc *RequestContext) InterceptNotification(method string, interceptor NotificationInterceptor) {
	if rc.notificationInterceptors == nil {
		rc.notificationInterceptors = make(map[string]NotificationInterceptor)
	}
	rc.notificationInterceptors[method] = interceptor
}

func (rc *RequestContext) StopInterceptingNotification(method string) {
	if rc.notificationInterceptors == nil {
		return
	}
	delete(rc.notificationInterceptors, method)
}

func (rc *RequestContext) Notify(method string, params interface{}) error {
	if rc.notificationInterceptors != nil {
		if ni, ok := rc.notificationInterceptors[method]; ok {
			return ni(method, params)
		}
	}
	return rc.Conn.Notify(rc.Ctx, method, params)
}

func (rc *RequestContext) RootClient() *itchio.Client {
	return rc.Client("<keyless>")
}

func (rc *RequestContext) ProfileClient(profileID int64) (*models.Profile, *itchio.Client) {
	if profileID == 0 {
		panic(errors.New("profileId must be non-zero"))
	}

	conn := rc.GetConn()
	defer rc.PutConn(conn)

	profile := models.ProfileByID(conn, profileID)
	if profile == nil {
		panic(errors.Errorf("Could not find profile %d", profileID))
	}

	if profile.APIKey == "" {
		panic(errors.Errorf("Profile %d lacks API key", profileID))
	}

	return profile, rc.Client(profile.APIKey)
}

func (rc *RequestContext) StartProgress() {
	rc.StartProgressWithTotalBytes(0)
}

func (rc *RequestContext) StartProgressWithTotalBytes(totalBytes int64) {
	rc.StartProgressWithInitialAndTotal(0.0, totalBytes)
}

func (rc *RequestContext) StartProgressWithInitialAndTotal(initialProgress float64, totalBytes int64) {
	if rc.tracker != nil {
		rc.Consumer.Warnf("Asked to start progress but already tracking progress!")
		return
	}

	trackerOpts := tracker.Opts{
		Value: initialProgress,
	}
	if totalBytes > 0 {
		trackerOpts.ByteAmount = &tracker.ByteAmount{Value: totalBytes}
	}
	rc.tracker = tracker.New(trackerOpts)
}

func (rc *RequestContext) EndProgress() {
	if rc.tracker != nil {
		rc.tracker = nil
	} else {
		rc.Consumer.Warnf("Asked to stop progress but wasn't tracking progress!")
	}
}

func (rc *RequestContext) GetConn() *sqlite.Conn {
	getCtx, cancel := context.WithTimeout(rc.Ctx, 3*time.Second)
	defer cancel()
	conn := rc.dbPool.Get(getCtx.Done())
	if conn == nil {
		panic(errors.WithStack(CodeDatabaseBusy))
	}

	conn.SetInterrupt(rc.Ctx.Done())
	return conn
}

func (rc *RequestContext) PutConn(conn *sqlite.Conn) {
	rc.dbPool.Put(conn)
}

func (rc *RequestContext) WithConn(f func(conn *sqlite.Conn)) {
	conn := rc.GetConn()
	defer rc.PutConn(conn)
	f(conn)
}

func (rc *RequestContext) WithConnBool(f func(conn *sqlite.Conn) bool) bool {
	conn := rc.GetConn()
	defer rc.PutConn(conn)
	return f(conn)
}

func (rc *RequestContext) WithConnString(f func(conn *sqlite.Conn) string) string {
	conn := rc.GetConn()
	defer rc.PutConn(conn)
	return f(conn)
}

type CancelFuncs struct {
	Funcs map[string]context.CancelFunc
}

func (cf *CancelFuncs) Add(id string, f context.CancelFunc) {
	cf.Funcs[id] = f
}

func (cf *CancelFuncs) Remove(id string) {
	delete(cf.Funcs, id)
}

func (cf *CancelFuncs) Call(id string) bool {
	if f, ok := cf.Funcs[id]; ok {
		f()
		delete(cf.Funcs, id)
		return true
	}

	return false
}
