package mansion

import (
	"net/http"
	"net/url"
	"testing"

	itchio "github.com/itchio/go-itchio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientTrimsAPIKeyWhitespace(t *testing.T) {
	ctx := &Context{
		HTTPClient: http.DefaultClient,
		apiAddress: "https://api.itch.io",
	}

	client := ctx.NewClient(" \tapi-key\r\n")
	assert.Equal(t, "api-key", client.Key)

	downloadURL := client.MakeBuildFileDownloadURL(itchio.MakeBuildFileDownloadURLParams{
		BuildID: 1,
		FileID:  2,
	})
	parsedURL, err := url.Parse(downloadURL)
	require.NoError(t, err)
	assert.Equal(t, "api-key", parsedURL.Query().Get("api_key"))
}
