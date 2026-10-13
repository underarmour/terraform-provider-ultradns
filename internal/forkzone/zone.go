// Package zone extends the pinned SDK only where the API's Valimail field is
// missing. All other DTOs and service operations use the released SDK directly.
package zone

import (
	"net/http"
	"net/url"

	"github.com/ultradns/ultradns-go-sdk/pkg/client"
	"github.com/ultradns/ultradns-go-sdk/pkg/errors"
	"github.com/ultradns/ultradns-go-sdk/pkg/task"
	sdk "github.com/ultradns/ultradns-go-sdk/pkg/zone"
)

const (
	Primary   = sdk.Primary
	Secondary = sdk.Secondary
	Alias     = sdk.Alias
)

type Properties = sdk.Properties
type SecondaryZone = sdk.SecondaryZone
type AliasZone = sdk.AliasZone
type NameServer = sdk.NameServer
type Tsig = sdk.Tsig
type RestrictIP = sdk.RestrictIP
type NotifyAddress = sdk.NotifyAddress
type NameServerIPList = sdk.NameServerIPList
type PrimaryNameServers = sdk.PrimaryNameServers
type RegistrarInfo = sdk.RegistrarInfo
type NameServersList = sdk.NameServersList
type TransferStatusDetails = sdk.TransferStatusDetails

type PrimaryZone struct {
	sdk.PrimaryZone
	ValimailMonitor *bool `json:"valimailMonitor,omitempty"`
}

type Zone struct {
	Properties          *Properties    `json:"properties,omitempty"`
	PrimaryCreateInfo   *PrimaryZone   `json:"primaryCreateInfo,omitempty"`
	SecondaryCreateInfo *SecondaryZone `json:"secondaryCreateInfo,omitempty"`
	AliasCreateInfo     *AliasZone     `json:"aliasCreateInfo,omitempty"`
	ChangeComment       string         `json:"changeComment,omitempty"`
}

type Response struct {
	sdk.Response
	ValimailMonitor *bool `json:"valimailMonitor,omitempty"`
}

type Service struct {
	*sdk.Service
	c *client.Client
}

func Get(c *client.Client) (*Service, error) {
	base, err := sdk.Get(c)
	if err != nil {
		return nil, err
	}
	return &Service{Service: base, c: c}, nil
}

func (z *Zone) sdkZone() *sdk.Zone {
	base := &sdk.Zone{
		Properties: z.Properties, SecondaryCreateInfo: z.SecondaryCreateInfo,
		AliasCreateInfo: z.AliasCreateInfo, ChangeComment: z.ChangeComment,
	}
	if z.PrimaryCreateInfo != nil {
		base.PrimaryCreateInfo = &z.PrimaryCreateInfo.PrimaryZone
	}
	return base
}

func (s *Service) CreateZone(z *Zone) (*http.Response, error) {
	if z.PrimaryCreateInfo == nil || z.PrimaryCreateInfo.ValimailMonitor == nil {
		return s.Service.CreateZone(z.sdkZone())
	}
	res, err := s.c.Do(http.MethodPost, "zones/", z, client.Target(&client.SuccessResponse{}))
	if err != nil {
		name := ""
		if z.Properties != nil {
			name = z.Properties.Name
		}
		return res, errors.CreateError("Zone", name, err)
	}
	// Match the SDK's asynchronous zone creation contract.
	if res.StatusCode == http.StatusAccepted {
		tasks, err := task.Get(s.c)
		if err != nil {
			return nil, err
		}
		if err := tasks.TaskWait(res.Header.Get("X-Task-Id"), 5, 10); err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (s *Service) ReadZone(name string) (*http.Response, *Response, error) {
	response := &Response{}
	res, err := s.c.Do(http.MethodGet, "zones/"+url.PathEscape(name), nil, client.Target(response))
	if err != nil {
		return res, nil, errors.ReadError("Zone", name, err)
	}
	return res, response, nil
}

func (s *Service) UpdateZone(name string, z *Zone) (*http.Response, error) {
	return s.Service.UpdateZone(name, z.sdkZone())
}

func (s *Service) PartialUpdateZone(name string, z *Zone) (*http.Response, error) {
	res, err := s.c.Do(http.MethodPatch, "zones/"+url.PathEscape(name), z, client.Target(&client.SuccessResponse{}))
	if err != nil {
		return res, errors.PartialUpdateError("Zone", name, err)
	}
	return res, nil
}
