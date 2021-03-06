package dl

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/terraform-providers/terraform-provider-outscale/osc"
	"github.com/terraform-providers/terraform-provider-outscale/osc/handler"
)

//DL the name of the api for url building
const DL = "directlink"

//Client manages the FCU API
type Client struct {
	client *osc.Client
	API    Service
}

// NewDLClient return a client to operate DL resources
func NewDLClient(config osc.Config) (*Client, error) {

	s := &v4.Signer{
		Credentials: credentials.NewStaticCredentials(config.Credentials.AccessKey,
			config.Credentials.SecretKey, ""),
	}

	u, err := url.Parse(fmt.Sprintf(osc.DefaultBaseURL, DL, config.Credentials.Region))
	if err != nil {
		return nil, err
	}

	config.Target = DL
	config.BaseURL = u
	config.UserAgent = osc.UserAgent
	config.Client = &http.Client{}

	c := osc.Client{
		Config:                config,
		Signer:                s,
		MarshalHander:         handler.URLEncodeMarshalHander,
		BuildRequestHandler:   handler.BuildURLEncodedRequest,
		UnmarshalHandler:      handler.UnmarshalDLHandler,
		UnmarshalErrorHandler: handler.UnmarshalJSONErrorHandler,
		SetHeaders:            handler.SetHeadersDL,
		BindBody:              handler.BindDL,
	}

	f := &Client{client: &c,
		API: Operations{client: &c},
	}
	return f, nil
}
