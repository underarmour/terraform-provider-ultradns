package webforward

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/ultradns/terraform-provider-ultradns/internal/service"
	"github.com/ultradns/ultradns-go-sdk/pkg/client"
)

func TestFindWebForwardByRequestToFollowsPagesAndNormalizesHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/authorization/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`)
		case r.Method == http.MethodGet && r.URL.Path == "/zones/example.com./webforwards":
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Query().Get("offset") == "1" {
				_, _ = io.WriteString(w, `{"webForwards":[{"guid":"target-guid","requestTo":"api.example.com"}],"resultInfo":{"totalCount":2,"offset":1,"returnedCount":1}}`)
				return
			}
			_, _ = io.WriteString(w, `{"webForwards":[{"guid":"other-guid","requestTo":"http://www.example.com"}],"resultInfo":{"totalCount":2,"offset":0,"returnedCount":1}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c, err := client.NewClient(client.Config{
		Username: "username",
		Password: "password",
		HostURL:  server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	services, err := service.NewService(c)
	if err != nil {
		t.Fatal(err)
	}

	rd := schema.TestResourceDataRaw(t, DataSourceWebForward().Schema, map[string]interface{}{
		"zone_name":  "example.com.",
		"request_to": "api.example.com",
	})

	webForward, err := findWebForward(services, "example.com.", rd)
	if err != nil {
		t.Fatal(err)
	}
	if webForward.GUID != "target-guid" {
		t.Fatalf("guid = %q, want target-guid", webForward.GUID)
	}

	if err := rd.Set("request_to", "www.example.com"); err != nil {
		t.Fatal(err)
	}
	webForward, err = findWebForward(services, "example.com.", rd)
	if err != nil {
		t.Fatal(err)
	}
	if webForward.GUID != "other-guid" {
		t.Fatalf("normalized request_to lookup guid = %q, want other-guid", webForward.GUID)
	}
}
