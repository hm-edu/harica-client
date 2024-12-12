package client

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"log/slog"

	"github.com/go-co-op/gocron/v2"
	"github.com/go-resty/resty/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hm-edu/harica/models"
	"github.com/pquerna/otp/totp"
)

const (
	BaseURL               = "https://cm.harica.gr"
	LoginPath             = "/api/User/Login"
	LoginPathTotp         = "/api/User/Login2FA"
	RevocationReasonsPath = "/api/Certificate/GetRevocationReasons"
	DomainValidationsPath = "/api/ServerCertificate/GetDomainValidations"
	RefreshInterval       = 15 * time.Minute
)

type Client struct {
	client       *resty.Client
	scheduler    gocron.Scheduler
	currentToken string
	debug        bool
}

type Option func(*Client)

func NewClient(user, password, totpSeed string, options ...Option) (*Client, error) {
	c := Client{}
	for _, option := range options {
		option(&c)
	}
	err := c.prepareClient(user, password, totpSeed)
	if err != nil {
		return nil, err
	}
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}
	_, err = s.NewJob(gocron.DurationJob(RefreshInterval), gocron.NewTask(func() {
		err := c.prepareClient(user, password, totpSeed)
		if err != nil {
			slog.Error("failed to prepare client", slog.Any("error", err))
			return
		}
	}))
	if err != nil {
		return nil, err
	}
	s.Start()
	c.scheduler = s
	return &c, nil
}

func WithDebug(debug bool) Option {
	return func(c *Client) {
		c.debug = debug
	}
}

func (c *Client) prepareClient(user, password, totpSeed string) error {
	renew := false

	if c.currentToken != "" {
		// Check JWT
		token, _, err := jwt.NewParser().ParseUnverified(c.currentToken, jwt.MapClaims{})
		if err != nil {
			return err
		}
		exp, err := token.Claims.GetExpirationTime()
		if err != nil {
			return err
		}
		if exp.Before(time.Now()) || exp.Before(time.Now().Add(RefreshInterval)) {
			renew = true
		}
	}
	if c.client == nil || c.currentToken == "" || renew {
		if totpSeed != "" {
			return c.loginTotp(user, password, totpSeed)
		} else {
			return c.login(user, password)
		}
	}
	return nil
}

func (c *Client) loginTotp(user, password, totpSeed string) error {
	r := resty.New()
	verificationToken, err := getVerificationToken(r)
	if err != nil {
		return err
	}
	otp, err := totp.GenerateCode(totpSeed, time.Now())
	if err != nil {
		return err
	}
	resp, err := r.
		R().SetHeaderVerbatim("RequestVerificationToken", verificationToken).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]string{"email": user, "password": password, "token": otp}).
		Post(BaseURL + LoginPathTotp)
	if err != nil {
		return err
	}
	tokenResp := strings.Trim(resp.String(), "\"")
	_, _, err = jwt.NewParser().ParseUnverified(tokenResp, jwt.MapClaims{})
	if err != nil {
		return err
	}
	c.currentToken = tokenResp
	r = r.SetHeaders(map[string]string{"Authorization": c.currentToken})
	token, err := getVerificationToken(r)
	if err != nil {
		return err
	}
	r = r.SetHeaderVerbatim("RequestVerificationToken", token).SetDebug(c.debug)
	c.client = r
	return nil
}

func (c *Client) login(user, password string) error {
	r := resty.New()
	verificationToken, err := getVerificationToken(r)
	if err != nil {
		return err
	}
	resp, err := r.
		R().SetHeaderVerbatim("RequestVerificationToken", verificationToken).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]string{"email": user, "password": password}).
		Post(BaseURL + LoginPath)
	if err != nil {
		return err
	}
	tokenResp := strings.Trim(resp.String(), "\"")
	_, _, err = jwt.NewParser().ParseUnverified(tokenResp, jwt.MapClaims{})
	if err != nil {
		return err
	}
	c.currentToken = tokenResp
	r = r.SetHeaders(map[string]string{"Authorization": c.currentToken})
	token, err := getVerificationToken(r)
	if err != nil {
		return err
	}
	r = r.SetHeaderVerbatim("RequestVerificationToken", token).SetDebug(c.debug)
	c.client = r
	return nil
}

func (c *Client) GetRevocationReasons() error {
	resp, err := c.client.R().Post(BaseURL + RevocationReasonsPath)
	if err != nil {
		return err
	}
	data := resp.String()
	fmt.Print(data)
	return nil
}

func (c *Client) GetDomainValidations() error {
	resp, err := c.client.R().Post(BaseURL + DomainValidationsPath)
	if err != nil {
		return err
	}
	fmt.Print(resp.String())
	return nil
}

type Domain struct {
	Domain string `json:"domain"`
}

func (c *Client) CheckMatchingOrganization(domains []string) ([]models.OrganizationResponse, error) {
	var domainDto []Domain
	for _, domain := range domains {
		domainDto = append(domainDto, Domain{Domain: domain})
	}
	var response []models.OrganizationResponse
	_, err := c.client.R().
		SetHeader("Content-Type", "application/json").
		SetResult(&response).SetBody(domainDto).
		Post(BaseURL + "/api/ServerCertificate/CheckMachingOrganization")
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (c *Client) GetCertificate(id string) (models.CertificateResponse, error) {
	var cert models.CertificateResponse
	_, err := c.client.R().
		SetResult(&cert).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]interface{}{"id": id}).
		Post(BaseURL + "/api/Certificate/GetCertificate")
	if err != nil {
		return cert, err
	}
	return cert, nil
}

func (c *Client) CheckDomainNames(domains []string) ([]models.DomainResponse, error) {
	domainDto := make([]Domain, 0)
	for _, domain := range domains {
		domainDto = append(domainDto, Domain{Domain: domain})
	}
	domainResp := make([]models.DomainResponse, 0)
	_, err := c.client.R().
		SetResult(&domainResp).
		SetHeader("Content-Type", "application/json").
		SetBody(domainDto).
		Post(BaseURL + "/api/ServerCertificate/CheckDomainNames")
	if err != nil {
		return nil, err
	}
	return domainResp, nil
}

func (c *Client) RequestCertificate(domains []models.DomainResponse, csr string, transactionType string) (models.CertificateRequestResponse, error) {
	domainJsonBytes, _ := json.Marshal(domains)
	domainJson := string(domainJsonBytes)
	var result models.CertificateRequestResponse
	_, err := c.client.R().
		SetHeader("Content-Type", "multipart/form-data").
		SetResult(&result).
		SetMultipartFormData(map[string]string{
			"domains":         domainJson,
			"domainsString":   domainJson,
			"csr":             csr,
			"isManualCsr":     "true",
			"consentSameKey":  "true",
			"transactionType": transactionType,
			"duration":        "1",
		}).
		Post(BaseURL + "/api/ServerCertificate/RequestServerCertificate")
	if err != nil {
		return result, err
	}
	return result, nil
}

func (c *Client) GetPendingReviews() ([]models.ReviewResponse, error) {
	var pending []models.ReviewResponse
	_, err := c.client.R().
		SetResult(&pending).
		SetHeader("Content-Type", "application/json").
		SetBody(models.ReviewRequest{
			StartIndex:     0,
			Status:         "Pending",
			FilterPostDTOs: []any{},
		}).
		Post(BaseURL + "/api/OrganizationValidatorSSL/GetSSLReviewableTransactions")
	if err != nil {
		return nil, err
	}
	return pending, nil
}

func (c *Client) ApproveRequest(id, message, value string) error {
	_, err := c.client.R().
		SetHeader("Content-Type", "multipart/form-data").
		SetMultipartFormData(map[string]string{
			"reviewId":        id,
			"isValid":         "true",
			"informApplicant": "true",
			"reviewMessage":   message,
			"reviewValue":     value,
		}).
		Post(BaseURL + "/api/OrganizationValidatorSSL/UpdateReviews")
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) Shutdown() error {
	err := c.scheduler.Shutdown()
	if err != nil {
		return err
	}
	return nil
}
