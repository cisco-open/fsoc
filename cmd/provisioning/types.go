// Copyright 2023 Cisco Systems, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provisioning

import "time"

const successState = "COMPLETED"

type TenantProvisioningRequest struct {
	Name         string       `json:"name,omitempty"`
	VanityUrl    string       `json:"vanityUrl,omitempty"`
	Account      Account      `json:"account,omitempty"`
	Organization Organization `json:"organization,omitempty"`
	User         User         `json:"user,omitempty"`
}

type TenantProvisioningResponse struct {
	TenantId   string `json:"tenantId"`
	WorkflowId string `json:"workflowId"`
}

type LicenseProvisioningRequest struct {
	LicenseId string     `json:"licenseId"`
	Revisions []Revision `json:"revisions"`
}

type LicenseProvisioningResponse struct {
	WorkflowId string `json:"workflowId"`
}

type SamlApplication struct {
	ExternalId  string `json:"externalId,omitempty"`
	SsoLoginUrl string `json:"ssoLoginUrl,omitempty"`
	SsoAcsUrl   string `json:"ssoAcsUrl,omitempty"`
	AudienceUri string `json:"audienceUri,omitempty"`
}

type TenantDetails struct {
	Id                    string          `json:"id"`
	Name                  string          `json:"name"`
	State                 string          `json:"state"`
	ProvisioningType      string          `json:"provisioningType"`
	VanityUrl             string          `json:"vanityUrl,omitempty"`
	Account               Account         `json:"account,omitempty"`
	Organization          Organization    `json:"organization,omitempty"`
	User                  User            `json:"user,omitempty"`
	SamlApplication       SamlApplication `json:"samlApplication,omitempty"`
	HasWorkflowInProgress bool            `json:"inProgressWorkflowExists"`
}

type WorkflowResponse struct {
	Id               string            `json:"id"`
	Type             string            `json:"type"`
	State            string            `json:"state"`
	StateDescription string            `json:"stateDescription,omitempty"`
	Context          map[string]string `json:"context,omitempty"`
	StateHistory     []stateHistory    `json:"stateHistory,omitempty"`
	Tenant           TenantDetails     `json:"tenant"`
}

type stateHistory struct {
	State            string `json:"state"`
	StateDescription string `json:"stateDescription,omitempty"`
	Retries          int32  `json:"retries"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
	CreatedAt        string `json:"createdAt"`
}
type TenantResponse struct {
	Id               string       `json:"id"`
	State            string       `json:"state"`
	ProvisioningType string       `json:"provisioningType"`
	Name             string       `json:"name"`
	VanityUrl        string       `json:"vanityUrl"`
	Region           string       `json:"region"`
	CellId           string       `json:"cellId"`
	CellHostName     string       `json:"cellHostName"`
	Account          Account      `json:"account"`
	Organization     Organization `json:"organization"`
	Workflows        []struct {
		Id               string    `json:"id"`
		Type             string    `json:"type"`
		State            string    `json:"state"`
		StateDescription string    `json:"stateDescription"`
		CreatedAt        time.Time `json:"createdAt"`
		UpdatedAt        time.Time `json:"updatedAt"`
	} `json:"workflows"`
}

type Account struct {
	ExternalId string `json:"externalId,omitempty"`
	Name       string `json:"name,omitempty"`
}

type Organization struct {
	ExternalId string `json:"externalId,omitempty"`
	Name       string `json:"name,omitempty"`
}

type User struct {
	ExternalId string `json:"externalId,omitempty"`
	Email      string `json:"email"`
	FirstName  string `json:"firstName,omitempty"`
	LastName   string `json:"lastName,omitempty"`
}

type Validity struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type Meter struct {
	MeterRef  string  `json:"meterRef"`
	UnitRatio float32 `json:"unitRatio"`
}

type UsageCycle struct {
	Period string `json:"period,omitempty"`
	Start  string `json:"start,omitempty"`
}

type Pool struct {
	Name       string     `json:"name"`
	Units      uint64     `json:"units"`
	Overage    int64      `json:"overage"`
	UsageCycle UsageCycle `json:"usageCycle"`
	Meters     []Meter    `json:"meters,omitempty"`
}

type Tier struct {
	TierRef string `json:"tierRef,omitempty"`
	Value   int8   `json:"value,omitempty"`
}

type Platform struct {
	LicenseType string `json:"licenseType,omitempty"`
	Pools       []Pool `json:"pools,omitempty"`
	Tiers       []Tier `json:"tiers,omitempty"`
}

type Solution struct {
	Name        string `json:"name,omitempty"`
	LicenseType string `json:"licenseType,omitempty"`
	Pools       []Pool `json:"pools,omitempty"`
}

type Entitlement struct {
	Platform  Platform   `json:"platform,omitempty"`
	Solutions []Solution `json:"solutions,omitempty"`
}

type DataRetention struct {
	DataType        string `json:"dataType,omitempty"`
	RetentionPeriod string `json:"period,omitempty"`
}

type Revision struct {
	RevisionId    string          `json:"revisionId"`
	Validity      Validity        `json:"validity"`
	DataRetention []DataRetention `json:"dataRetention,omitempty"`
	Entitlement   Entitlement     `json:"entitlements,omitempty"`
}
