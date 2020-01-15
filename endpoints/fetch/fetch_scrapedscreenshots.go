package fetch

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/itchio/butler/butlerd"
	"github.com/pkg/errors"
)

func FetchScrapedScreenshots(rc *butlerd.RequestContext, params butlerd.FetchScrapedScreenshotsParams) (*butlerd.FetchScrapedScreenshotsResult, error) {
	consumer := rc.Consumer
	res := &butlerd.FetchScrapedScreenshotsResult{}

	game := LazyFetchGame(rc, params.GameID)
	if game == nil {
		return res, nil
	}

	consumer.Infof("Scraping %q", game.URL)
	hres, err := rc.HTTPClient.Get(game.URL)
	if err != nil {
		return nil, err
	}

	if hres.StatusCode != 200 {
		return nil, errors.Errorf("HTTP code %d", hres.StatusCode)
	}

	defer hres.Body.Close()
	doc, err := goquery.NewDocumentFromReader(hres.Body)
	if err != nil {
		return nil, err
	}

	doc.Find(".screenshot_list img.screenshot").Each(func(i int, s *goquery.Selection) {
		attrs := s.Get(0).Attr

		found2x := false
		for _, attr := range attrs {
			if attr.Key == "srcset" {
				consumer.Infof("Found srcset %q", attr.Val)
				entries := strings.Split(attr.Val, ",")
				for _, entry := range entries {
					tokens := strings.Split(strings.TrimSpace(entry), " ")
					url := tokens[0]
					size := tokens[1]
					if size == "2x" {
						consumer.Infof("Found 2x screenshot %q", url)
						found2x = true
						res.Screenshots = append(res.Screenshots, url)
					}
				}
			}
		}

		if !found2x {
			for _, attr := range attrs {
				if attr.Key == "src" {
					url := attr.Val
					consumer.Infof("Found screenshot %q", url)
					res.Screenshots = append(res.Screenshots, url)
				}
			}
		}
	})

	return res, nil
}
