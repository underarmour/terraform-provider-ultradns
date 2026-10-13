package zone

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	zone "github.com/ultradns/terraform-provider-ultradns/internal/forkzone"
	"github.com/ultradns/terraform-provider-ultradns/internal/service"
	"github.com/ultradns/ultradns-go-sdk/pkg/client"
)

func primaryConfig(enabled interface{}, inherit string) map[string]interface{} {
	block := map[string]interface{}{"create_type": "NEW", "inherit": inherit}
	if enabled != nil {
		block["valimail_monitor"] = enabled
	}
	return map[string]interface{}{
		"name": "example.com.", "account_name": "account", "type": zone.Primary,
		"primary_create_info": []interface{}{block},
	}
}

func TestValimailSchemaAndExpansion(t *testing.T) {
	resource := ResourceZone()
	if err := resource.InternalValidate(resource.Schema, true); err != nil {
		t.Fatal(err)
	}
	field := resource.Schema["primary_create_info"].Elem.(*schema.Resource).Schema["valimail_monitor"]
	if !field.Optional || field.ForceNew || field.Default != false {
		t.Fatalf("unexpected Valimail schema: %+v", field)
	}
	for _, value := range []interface{}{nil, false, true} {
		rd := schema.TestResourceDataRaw(t, resource.Schema, primaryConfig(value, "NONE"))
		payload := newZone(rd)
		if payload.PrimaryCreateInfo.ValimailMonitor == nil {
			t.Fatal("primary create must explicitly send Valimail, including false")
		}
		want := value == true
		if *payload.PrimaryCreateInfo.ValimailMonitor != want {
			t.Fatalf("input %v: got %v, want %v", value, *payload.PrimaryCreateInfo.ValimailMonitor, want)
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		var body map[string]interface{}
		if err := json.Unmarshal(encoded, &body); err != nil {
			t.Fatal(err)
		}
		if got := body["primaryCreateInfo"].(map[string]interface{})["valimailMonitor"]; got != want {
			t.Fatalf("serialized %s: got %v, want %v", encoded, got, want)
		}
	}
	for _, kind := range []string{zone.Secondary, zone.Alias} {
		config := primaryConfig(true, "NONE")
		config["type"] = kind
		rd := schema.TestResourceDataRaw(t, resource.Schema, config)
		if payload := newZone(rd); payload.PrimaryCreateInfo != nil {
			t.Fatalf("%s must not send primary/Valimail settings", kind)
		}
	}
}

func TestValimailRefresh(t *testing.T) {
	for _, value := range []interface{}{nil, false, true} {
		rd := schema.TestResourceDataRaw(t, resourceZoneSchema(), primaryConfig(true, "NONE"))
		response := &zone.Response{}
		if value != nil {
			enabled := value.(bool)
			response.ValimailMonitor = &enabled
		}
		if err := flattenPrimaryZone(response, rd); err != nil {
			t.Fatal(err)
		}
		if got := rd.Get("primary_create_info.0.valimail_monitor"); got != (value == true) {
			t.Fatalf("response %v: got %v", value, got)
		}
		if got := rd.Get("primary_create_info.0.create_type"); got != "NEW" {
			t.Fatalf("refresh discarded create_type: %v", got)
		}
	}
}

func zoneTestService(t *testing.T, handler http.HandlerFunc) *service.Service {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/authorization/token" {
			_, _ = io.WriteString(w, `{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`)
			return
		}
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	c, err := client.NewClient(client.Config{Username: "test", Password: "test", HostURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	services, err := service.NewService(c)
	if err != nil {
		t.Fatal(err)
	}
	return services
}

func TestValimailCreateAndRead(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "disabled", true: "enabled"}[enabled], func(t *testing.T) {
			var methods []string
			services := zoneTestService(t, func(w http.ResponseWriter, r *http.Request) {
				methods = append(methods, r.Method)
				switch r.Method {
				case http.MethodPost:
					var payload zone.Zone
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Error(err)
					}
					if payload.PrimaryCreateInfo.ValimailMonitor == nil || *payload.PrimaryCreateInfo.ValimailMonitor != enabled {
						t.Errorf("unexpected create payload: %+v", payload)
					}
					w.WriteHeader(http.StatusCreated)
					_, _ = io.WriteString(w, `{"message":"created"}`)
				case http.MethodGet:
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"properties":      map[string]interface{}{"name": "example.com.", "type": zone.Primary, "accountName": "account"},
						"valimailMonitor": enabled,
					})
				default:
					t.Errorf("unexpected method: %s", r.Method)
				}
			})
			rd := schema.TestResourceDataRaw(t, resourceZoneSchema(), primaryConfig(enabled, "NONE"))
			if diags := resourceZoneCreate(context.Background(), rd, services); diags.HasError() {
				t.Fatal(diags)
			}
			if rd.Id() != "example.com." || rd.Get("primary_create_info.0.valimail_monitor") != enabled {
				t.Fatalf("unexpected state: id=%s enabled=%v", rd.Id(), rd.Get("primary_create_info.0.valimail_monitor"))
			}
			if !reflect.DeepEqual(methods, []string{"POST", "GET"}) {
				t.Fatalf("unexpected requests: %v", methods)
			}
		})
	}
}

func TestValimailUpdates(t *testing.T) {
	for _, tc := range []struct {
		name       string
		old, next  bool
		oldInherit string
		newInherit string
		methods    []string
	}{
		{"enable", false, true, "NONE", "NONE", []string{"PATCH", "GET"}},
		{"disable", true, false, "NONE", "NONE", []string{"PATCH", "GET"}},
		{"unrelated keeps enabled", true, true, "NONE", "ALL", []string{"PUT", "PATCH", "GET"}},
		{"unrelated keeps disabled", false, false, "NONE", "ALL", []string{"PUT", "PATCH", "GET"}},
		{"both", true, false, "NONE", "ALL", []string{"PUT", "PATCH", "GET"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource := ResourceZone()
			old := schema.TestResourceDataRaw(t, resource.Schema, primaryConfig(tc.old, tc.oldInherit))
			old.SetId("example.com.")
			diff, err := resource.Diff(context.Background(), old.State(), terraform.NewResourceConfigRaw(primaryConfig(tc.next, tc.newInherit)), nil)
			if err != nil {
				t.Fatal(err)
			}
			if diff.RequiresNew() {
				t.Fatal("Valimail update must not replace the zone")
			}
			rd, err := schema.InternalMap(resource.Schema).Data(old.State(), diff)
			if err != nil {
				t.Fatal(err)
			}
			var methods []string
			services := zoneTestService(t, func(w http.ResponseWriter, r *http.Request) {
				methods = append(methods, r.Method)
				if r.URL.Path != "/zones/example.com." {
					t.Errorf("unexpected update path: %s", r.URL.Path)
				}
				switch r.Method {
				case http.MethodPatch:
					var payload map[string]interface{}
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Error(err)
					}
					want := map[string]interface{}{"primaryCreateInfo": map[string]interface{}{"valimailMonitor": tc.next}}
					if !reflect.DeepEqual(payload, want) {
						t.Errorf("PATCH = %#v, want %#v", payload, want)
					}
					_, _ = io.WriteString(w, `{"message":"updated"}`)
				case http.MethodPut:
					var payload zone.Zone
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Error(err)
					}
					if payload.PrimaryCreateInfo.ValimailMonitor != nil || payload.PrimaryCreateInfo.Inherit != tc.newInherit {
						t.Errorf("unexpected PUT primary settings: %+v", payload.PrimaryCreateInfo)
					}
					_, _ = io.WriteString(w, `{"message":"updated"}`)
				case http.MethodGet:
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"properties":      map[string]interface{}{"name": "example.com.", "type": zone.Primary, "accountName": "account"},
						"valimailMonitor": tc.next, "inherit": tc.newInherit,
					})
				default:
					t.Errorf("unexpected method: %s", r.Method)
				}
			})
			if diags := resourceZoneUpdate(context.Background(), rd, services); diags.HasError() {
				t.Fatal(diags)
			}
			if !reflect.DeepEqual(methods, tc.methods) {
				t.Fatalf("requests %v, want %v", methods, tc.methods)
			}
			if rd.Id() != old.Id() || rd.Get("primary_create_info.0.valimail_monitor") != tc.next {
				t.Fatalf("unexpected update state: id=%s enabled=%v", rd.Id(), rd.Get("primary_create_info.0.valimail_monitor"))
			}
		})
	}
}
