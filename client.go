package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"log/slog"

	"github.com/go-co-op/gocron/v2"
	"github.com/go-resty/resty/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hm-edu/harica-client/models"
	"github.com/pquerna/otp/totp"
)

const (
	BaseURL         = "https://cm.harica.gr"
	LoginPath       = "/api/User/Login"
	LoginPathTotp   = "/api/User/Login2FA"
	RefreshInterval = 15 * time.Minute
)

type Client struct {
	client       *resty.Client
	scheduler    gocron.Scheduler
	currentToken string
}

func NewClient(user, password, totpSeed string) (*Client, error) {
	c := Client{}
	err := c.prepareClient(user, password, totpSeed)
	if err != nil {
		return nil, err
	}
	s, err := gocron.NewScheduler()
	if err != nil {
		slog.Error("failed to create scheduler", slog.Any("error", err))
		return nil, err
	}
	job, err := s.NewJob(gocron.DurationJob(RefreshInterval), gocron.NewTask(func() {
		c.prepareClient(user, password, totpSeed)
	}))
	if err != nil {
		slog.Error("failed to create job", slog.Any("error", err))
		return nil, err
	}
	slog.Info("added job", slog.Any("job", job))
	s.Start()
	c.scheduler = s
	return &c, nil
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
		// handle error
	}
	tokenResp := strings.Trim(resp.String(), "\"")
	_, _, err = jwt.NewParser().ParseUnverified(tokenResp, jwt.MapClaims{})
	if err != nil {
		return err
	}
	c.currentToken = tokenResp
	r = r.SetHeaders(map[string]string{"Authorization": c.currentToken})
	r.SetPreRequestHook(func(c *resty.Client, r *http.Request) error {
		cli := resty.New().SetCookieJar(c.GetClient().Jar)
		verificationToken, err := getVerificationToken(cli)
		if err != nil {
			slog.Error("failed to get verification token", slog.Any("error", err))
			return err
		}
		r.Header["RequestVerificationToken"] = []string{verificationToken}
		return nil
	})
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
	c.currentToken = strings.Trim(resp.String(), "\"")
	r = r.SetHeaders(map[string]string{"Authorization": c.currentToken})
	r.SetPreRequestHook(func(c *resty.Client, r *http.Request) error {
		cli := resty.New().SetCookieJar(c.GetClient().Jar)
		verificationToken, err := getVerificationToken(cli)
		if err != nil {
			slog.Error("failed to get verification token", slog.Any("error", err))
			return err
		}
		r.Header.Add("RequestVerificationToken", verificationToken)
		return nil
	})
	c.client = r
	return nil
}

func (c *Client) GetRevocationReasons() error {
	resp, err := c.client.R().Post(BaseURL + "/api/Certificate/GetRevocationReasons")
	if err != nil {
		return err
	}
	// handle response
	data := resp.String()
	fmt.Print(data)
	return nil
}

func (c *Client) GetDomainValidations() error {
	resp, err := c.client.R().Post(BaseURL + "/api/ServerCertificate/GetDomainValidations")
	if err != nil {
		return err
	}
	// handle response
	fmt.Print(resp.String())
	return nil
}

type Domain struct {
	Domain string `json:"domain"`
}

func (c *Client) CheckMatchingOrganization(domains []string) ([]models.OrganizationResponse, error) {
	domainDto := make([]Domain, 0)
	for _, domain := range domains {
		domainDto = append(domainDto, Domain{Domain: domain})
	}
	response := []models.OrganizationResponse{}
	_, err := c.client.R().SetHeader("Content-Type", "application/json").SetResult(&response).SetBody(domains).Post(BaseURL + "/api/ServerCertificate/CheckMachingOrganization")
	if err != nil {
		// handle error
		return nil, err
	}
	// handle response
	return response, nil
}

func (c *Client) CheckDomainNames(domains []string) ([]models.DomainResponse, error) {
	domainDto := make([]Domain, 0)
	for _, domain := range domains {
		domainDto = append(domainDto, Domain{Domain: domain})
	}
	domainResp := make([]models.DomainResponse, 0)
	_, err := c.client.R().SetResult(&domainResp).SetHeader("Content-Type", "application/json").SetBody(domainDto).Post(BaseURL + "/api/ServerCertificate/CheckDomainNames")
	if err != nil {
		return nil, err
	}
	// handle response
	return domainResp, nil
}

func (c *Client) RequestCertificate() {

}

func (c *Client) RevokeCertificate() {

}

func (c *Client) GetPendingReviews() ([]models.ReviewResponse, error) {
	pending := []models.ReviewResponse{}
	_, err := c.client.R().
		SetDebug(true).
		SetResult(&pending).
		SetHeader("Content-Type", "application/json").
		SetBody(models.ReviewRequest{StartIndex: 0, Status: "Pending", FilterPostDTOs: []any{}}).
		Post(BaseURL + "/api/OrganizationValidatorSSL/GetSSLReviewableTransactions")
	if err != nil {
		return nil, err
	}
	return pending, nil
}

func (c *Client) ApproveRequest(id, message, value string) {

	c.client.R().
		SetDebug(true).
		SetHeader("Content-Type", "multipart/form-data").
		SetBody(map[string]string{"reviewId": id, "isValid": "true", "informApplicant": "true", "reviewMessage": message, "reviewValue": value}).
		Post(BaseURL + "/api/OrganizationValidatorSSL/UpdateReviews")

}

func (c *Client) Shutdown() {
	err := c.scheduler.Shutdown()
	if err != nil {
		// handle error
	}
}
