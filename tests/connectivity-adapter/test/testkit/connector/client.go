package connector

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"testing"

	"github.com/kyma-incubator/compass/tests/connectivity-adapter/test/testkit/director"

	"github.com/stretchr/testify/require"
)

const (
	ApplicationHeader = "Application"
	GroupHeader       = "Group"
	TenantHeader      = "Tenant"
	Tenant            = "testkit-tenant"
	Extensions        = ""
	KeyAlgorithm      = "rsa2048"
)

type ConnectorClient interface {
	CreateToken(t *testing.T) TokenResponse
	GetInfo(t *testing.T, url string) (*InfoResponse, *Error)
	CreateCertChain(t *testing.T, csr, url string) (*CrtResponse, *Error)
}

type connectorClient struct {
	httpClient     *http.Client
	directorClient director.Client
	appID          string
	tenant         string
}

func NewConnectorClient(directorClient director.Client, appID, tenant string, skipVerify bool) ConnectorClient {
	client := NewHttpClient(skipVerify)

	return connectorClient{
		httpClient:     client,
		directorClient: directorClient,
		appID:          appID,
		tenant:         tenant,
	}
}

func NewHttpClient(skipVerify bool) *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipVerify},
	}
	client := &http.Client{Transport: tr}
	return client
}

func (cc connectorClient) CreateToken(t *testing.T) TokenResponse {
	url, token, err := cc.directorClient.GetOneTimeTokenUrl(cc.appID)
	require.NoError(t, err)
	tokenResponse := TokenResponse{
		URL:   url,
		Token: token,
	}

	return tokenResponse
}

func (cc connectorClient) GetInfo(t *testing.T, url string) (*InfoResponse, *Error) {
	request := getRequestWithHeaders(t, cc.tenant, url)

	response, err := cc.httpClient.Do(request)
	require.NoError(t, err)
	defer func() {
		err := response.Body.Close()
		require.NoError(t, err)
	}()

	if response.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(t, response)
	}

	require.Equal(t, http.StatusOK, response.StatusCode)

	infoResponse := &InfoResponse{}

	err = json.NewDecoder(response.Body).Decode(&infoResponse)
	require.NoError(t, err)

	return infoResponse, nil
}

func (cc connectorClient) CreateCertChain(t *testing.T, csr, url string) (*CrtResponse, *Error) {
	body, err := json.Marshal(CsrRequest{Csr: csr})
	require.NoError(t, err)

	request, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	require.NoError(t, err)
	request.Close = true
	request.Header.Add("Content-Type", "application/json")

	response, err := cc.httpClient.Do(request)
	require.NoError(t, err)
	defer func() {
		err := response.Body.Close()
		require.NoError(t, err)
	}()

	if response.StatusCode != http.StatusCreated {
		return nil, parseErrorResponse(t, response)
	}

	require.Equal(t, http.StatusCreated, response.StatusCode)

	crtResponse := &CrtResponse{}

	err = json.NewDecoder(response.Body).Decode(&crtResponse)
	require.NoError(t, err)

	return crtResponse, nil
}

func getRequestWithHeaders(t *testing.T, tenant, url string) *http.Request {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)

	request.Header.Set("Tenant", tenant)
	request.Close = true

	return request
}

func parseErrorResponse(t *testing.T, response *http.Response) *Error {
	logResponse(t, response)
	errorResponse := ErrorResponse{}
	err := json.NewDecoder(response.Body).Decode(&errorResponse)
	require.NoError(t, err)

	return &Error{response.StatusCode, errorResponse}
}

func logResponse(t *testing.T, resp *http.Response) {
	respDump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Logf("failed to dump response, %s", err)
	}

	reqDump, err := httputil.DumpRequest(resp.Request, true)
	if err != nil {
		t.Logf("failed to dump request, %s", err)
	}

	if err == nil {
		t.Logf("\n--------------------------------\n%s\n--------------------------------\n%s\n--------------------------------", reqDump, respDump)
	}
}
