package templatemailer

import (
	"net/http"
	"net/url"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supabase/auth/internal/conf"
	"github.com/supabase/auth/internal/utilities"
)

var urlRegexp = regexp.MustCompile(`^https?://[^/]+`)

func enforceRelativeURL(url string) string {
	return urlRegexp.ReplaceAllString(url, "")
}

func TestGetPath(t *testing.T) {
	params := emailParams{
		Token:      "token",
		Type:       "signup",
		RedirectTo: "https://example.com",
	}
	cases := []struct {
		SiteURL  string
		Path     string
		Params   *emailParams
		Expected string
	}{
		{
			SiteURL:  "https://test.example.com",
			Path:     "/templates/confirm.html",
			Params:   nil,
			Expected: "https://test.example.com/templates/confirm.html",
		},
		{
			SiteURL:  "https://test.example.com/removedpath",
			Path:     "/templates/confirm.html",
			Params:   nil,
			Expected: "https://test.example.com/templates/confirm.html",
		},
		{
			SiteURL:  "https://test.example.com/",
			Path:     "/trailingslash/",
			Params:   nil,
			Expected: "https://test.example.com/trailingslash/",
		},
		{
			SiteURL:  "https://test.example.com",
			Path:     "f",
			Params:   &params,
			Expected: "https://test.example.com/f?token=token&type=signup&redirect_to=https://example.com",
		},
		{
			SiteURL:  "https://test.example.com",
			Path:     "",
			Params:   &params,
			Expected: "https://test.example.com?token=token&type=signup&redirect_to=https://example.com",
		},
	}

	for _, c := range cases {
		u, err := url.ParseRequestURI(c.SiteURL)
		assert.NoError(t, err, "error parsing URI request")

		path, err := getPath(c.Path, c.Params)

		assert.NoError(t, err)
		assert.Equal(t, c.Expected, u.ResolveReference(path).String())
	}
}

// TestGetPathRedirectToRoundTrip checks that the redirect_to a caller asked
// for is the redirect_to the server reads back when the emailed link is
// followed, i.e. that getPath does not alter or drop it.
func TestGetPathRedirectToRoundTrip(t *testing.T) {
	config := conf.GlobalConfiguration{
		SiteURL:      "https://app.example.com",
		URIAllowList: []string{"https://app.example.com/**", "myapp://callback/**"},
		JWT:          conf.JWTConfiguration{Secret: "testsecret"},
	}
	require.NoError(t, config.ApplyDefaults())

	cases := []struct {
		name       string
		redirectTo string
	}{
		{"plain path", "https://app.example.com/dashboard"},
		{"query string", "https://app.example.com/dashboard?tab=1"},
		{"query string with ampersand", "https://app.example.com/dashboard?tab=1&sort=asc"},
		{"fragment", "https://app.example.com/dashboard#section"},
		{"plus sign in path", "https://app.example.com/files/a+b.pdf"},
		{"plus sign in custom scheme", "myapp://callback/user+tag"},
		{"semicolon in path", "https://app.example.com/files/a;b"},
		{"literal percent sign", "https://app.example.com/p/100%25"},
		{"percent-encoded plus", "https://app.example.com/search/a%2Bb"},
		{"percent-encoded space", "https://app.example.com/teams/My%20Team"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path, err := getPath("/verify", &emailParams{
				Token:      "token",
				Type:       "signup",
				RedirectTo: c.redirectTo,
			})
			require.NoError(t, err)

			link := "https://auth.example.com/auth/v1/verify?" + path.RawQuery
			req, err := http.NewRequest(http.MethodGet, link, nil)
			require.NoError(t, err)

			assert.Equal(t, c.redirectTo, utilities.GetReferrer(req, &config))
		})
	}
}

func TestRelativeURL(t *testing.T) {
	cases := []struct {
		URL      string
		Expected string
	}{
		{"https://test.example.com", ""},
		{"http://test.example.com", ""},
		{"test.example.com", "test.example.com"},
		{"/some/path#fragment", "/some/path#fragment"},
	}

	for _, c := range cases {
		res := enforceRelativeURL(c.URL)
		assert.Equal(t, c.Expected, res, c.URL)
	}
}
