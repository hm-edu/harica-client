package models

type DomainResponse struct {
	Domain         string `json:"domain"`
	IsValid        bool   `json:"isValid"`
	IncludeWWW     bool   `json:"includeWWW"`
	ErrorMessage   string `json:"errorMessage"`
	WarningMessage string `json:"warningMessage"`
	IsPrevalidated bool   `json:"isPrevalidated"`
	IsWildcard     bool   `json:"isWildcard"`
	IsFreeDomain   bool   `json:"isFreeDomain"`
	IsFreeDomainDV bool   `json:"isFreeDomainDV"`
	IsFreeDomainEV bool   `json:"isFreeDomainEV"`
	CanRequestOV   bool   `json:"canRequestOV"`
	CanRequestEV   bool   `json:"canRequestEV"`
}

type OrganizationResponse struct {
	ID                            string `json:"id"`
	OrganizationName              string `json:"organizationName"`
	OrganizationUnitName          string `json:"organizationUnitName"`
	State                         string `json:"state"`
	Locality                      string `json:"locality"`
	Country                       string `json:"country"`
	Dn                            string `json:"dn"`
	OrganizationNameLocalized     string `json:"organizationNameLocalized"`
	OrganizationUnitNameLocalized string `json:"organizationUnitNameLocalized"`
	StateLocalized                string `json:"stateLocalized"`
	LocalityLocalized             string `json:"localityLocalized"`
	OrganizationIdentifier        string `json:"organizationIdentifier"`
	IsBaseDomain                  bool   `json:"isBaseDomain"`
	JurisdictionCountry           string `json:"jurisdictionCountry"`
	JurisdictionState             string `json:"jurisdictionState"`
	JurisdictionLocality          string `json:"jurisdictionLocality"`
	BusinessCategory              string `json:"businessCategory"`
	Serial                        string `json:"serial"`
	GroupDomains                  any    `json:"groupDomains"`
}
