package awssdkhelper

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	ThcompUtility "github.com/thcomp/GoLang_Utility"
)

type HttpRequestHandler func(r *http.Request, w http.ResponseWriter)
type ApiGwProxyHandler1 func(event *events.APIGatewayProxyRequest) error
type ApiGwProxyHandler2 func(context events.APIGatewayProxyRequestContext, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error)
type ApiGwV2HttpHandler1 func(event *events.APIGatewayV2HTTPRequest) error
type ApiGwV2HttpHandler2 func(context *events.APIGatewayV2HTTPRequestContext, event *events.APIGatewayV2HTTPRequest) (*events.APIGatewayV2HTTPResponse, error)
type ApiGwWebsocketHandler func(context *events.APIGatewayWebsocketProxyRequestContext, event *events.APIGatewayWebsocketProxyRequest) error
type LambdaFunctionURLHandler1 func(event *events.LambdaFunctionURLRequest) error
type LambdaFunctionURLHandler2 func(context *events.LambdaFunctionURLRequestContext, event *events.LambdaFunctionURLRequest) (*events.LambdaFunctionURLResponse, error)
type SimpleNotificationServiceHandler func(event *events.SNSEvent) error
type SimpleQueueServiceHandler func(event *events.SQSEvent) error
type SimpleEmailEventHandler func(event *events.SimpleEmailEvent) error

type httpHandlerInfo struct {
	userHttpRequestHandler               HttpRequestHandler
	userApiGwProxyHandler1               ApiGwProxyHandler1
	userApiGwProxyHandler2               ApiGwProxyHandler2
	userApiGwV2HttpHandler1              ApiGwV2HttpHandler1
	userApiGwV2HttpHandler2              ApiGwV2HttpHandler2
	userApiGwWebsocketHandler            ApiGwWebsocketHandler
	userLambdaFunctionURLHandler1        LambdaFunctionURLHandler1
	userLambdaFunctionURLHandler2        LambdaFunctionURLHandler2
	userSimpleNotificationServiceHandler SimpleNotificationServiceHandler
	userSimpleQueueServiceHandler        SimpleQueueServiceHandler
	userSimpleEmailEventHandler          SimpleEmailEventHandler
}

func (info *httpHandlerInfo) Handler1(context context.Context, event interface{}) (err error) {
	if awsEvent, assertionOK := event.(*events.SimpleEmailEvent); assertionOK {
		if info.userSimpleEmailEventHandler != nil {
			err = info.userSimpleEmailEventHandler(awsEvent)
		} else {
			err = fmt.Errorf("not set userSimpleEmailEventHandler")
		}
	} else if awsEvent, assertionOK := event.(*events.SNSEvent); assertionOK {
		if info.userSimpleNotificationServiceHandler != nil {
			err = info.userSimpleNotificationServiceHandler(awsEvent)
		} else {
			err = fmt.Errorf("not set userSimpleNotificationServiceHandler")
		}
	} else if awsEvent, assertionOK := event.(*events.SQSEvent); assertionOK {
		if info.userSimpleQueueServiceHandler != nil {
			err = info.userSimpleQueueServiceHandler(awsEvent)
		} else {
			err = fmt.Errorf("not set userSimpleQueueServiceHandler")
		}
	} else {
		if info.userApiGwProxyHandler1 != nil {

		} else if info.userApiGwV2HttpHandler1 != nil {

		} else if info.userLambdaFunctionURLHandler1 != nil {

		}
	}

	return err
}

func (info *httpHandlerInfo) Handler2(context context.Context, event interface{}) (out interface{}, err error) {
	if awsEvent, assertionOK := event.(*events.APIGatewayProxyRequest); assertionOK {
		if info.userHttpRequestHandler != nil {
			if httpReq, exchangeErr := FromAPIGatewayProxyRequest2HttpRequest(awsEvent); exchangeErr == nil {
				httpRes := ThcompUtility.NewHttpResponseHelper()
				info.userHttpRequestHandler(httpReq, httpRes)
				out, err = FromHttpResponse2APIGatewayProxyResponse(httpRes.ExportHttpResponse())
			} else {
				err = exchangeErr
			}
		} else if info.userApiGwProxyHandler2 != nil {
			out, err = info.userApiGwProxyHandler2(awsEvent.RequestContext, awsEvent)
		} else {
			err = fmt.Errorf("not set userApiGwProxyHandler")
		}
	} else if awsEvent, assertionOK := event.(*events.APIGatewayV2HTTPRequest); assertionOK {
		if info.userHttpRequestHandler != nil {
			if httpReq, exchangeErr := FromAPIGatewayV2HTTPRequest2HttpRequest(awsEvent); exchangeErr == nil {
				httpRes := ThcompUtility.NewHttpResponseHelper()
				info.userHttpRequestHandler(httpReq, httpRes)
				out, err = FromHttpResponse2APIGatewayV2HTTPResponse(httpRes.ExportHttpResponse())
			} else {
				err = exchangeErr
			}
		} else if info.userApiGwV2HttpHandler2 != nil {
			out, err = info.userApiGwV2HttpHandler2(&awsEvent.RequestContext, awsEvent)
		} else {
			err = fmt.Errorf("not set userApiGwV2HttpHandler")
		}
	} else if awsEvent, assertionOK := event.(*events.LambdaFunctionURLRequest); assertionOK {
		if info.userHttpRequestHandler != nil {
			if httpReq, exchangeErr := FromLambdaFunctionURLRequest2HttpRequest(awsEvent); exchangeErr == nil {
				httpRes := ThcompUtility.NewHttpResponseHelper()
				info.userHttpRequestHandler(httpReq, httpRes)
				out, err = FromHttpResponse2LambdaFunctionURLResponse(httpRes.ExportHttpResponse())
			} else {
				err = exchangeErr
			}
		} else if info.userLambdaFunctionURLHandler2 != nil {
			out, err = info.userLambdaFunctionURLHandler2(&awsEvent.RequestContext, awsEvent)
		}
	} else {
		err = fmt.Errorf("unknown format event")
	}

	return
}

func FromAPIGatewayProxyRequest2HttpRequest(from *events.APIGatewayProxyRequest) (req *http.Request, err error) {
	req = &http.Request{
		Method: from.HTTPMethod,
	}

	if from.IsBase64Encoded {
		if decodedBody, decodeErr := base64.StdEncoding.DecodeString(from.Body); decodeErr == nil {
			req.Body = ioutil.NopCloser(bytes.NewReader(decodedBody))
			req.ContentLength = int64(len(decodedBody))
		} else {
			err = decodeErr
		}
	} else {
		req.Body = ioutil.NopCloser(bytes.NewReader([]byte(from.Body)))
		req.ContentLength = int64(len(from.Body))
	}

	if err == nil {
		if req.Header == nil {
			req.Header = http.Header{}
		}

		for key, value := range from.Headers {
			req.Header.Add(key, value)
		}
		for key, values := range from.MultiValueHeaders {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}

		baseURLbuilder := ThcompUtility.StringBuilder{}
		baseURLbuilder.Append("http://localhost/")
		if len(from.Path) > 0 {
			from.Path = strings.TrimPrefix(from.Path, "/")
			baseURLbuilder.Append(from.Path)
		}

		if len(from.QueryStringParameters) > 0 || len(from.MultiValueQueryStringParameters) > 0 {
			queries := []string{}
			for key, value := range from.QueryStringParameters {
				queries = append(queries, key+"="+value)
			}
			for key, values := range from.MultiValueQueryStringParameters {
				for _, value := range values {
					queries = append(queries, key+"="+value)
				}
			}

			queriesText := strings.Join(queries, "&")
			baseURLbuilder.Append("?").Append(queriesText)
		}

		if tempURL, parseErr := url.Parse(baseURLbuilder.String()); parseErr == nil {
			req.URL = tempURL
		} else {
			err = parseErr
		}
	}

	return
}

func FromHttpResponse2APIGatewayProxyResponse(res *http.Response) (to *events.APIGatewayProxyResponse, err error) {
	to = &events.APIGatewayProxyResponse{
		StatusCode: res.StatusCode,
	}

	mimeType := ``
	for key, values := range res.Header {
		if len(values) > 1 {
			if to.MultiValueHeaders == nil {
				to.MultiValueHeaders = map[string][]string{}
			}
			to.MultiValueHeaders[key] = values
		} else if len(values) == 1 {
			if to.Headers == nil {
				to.Headers = map[string]string{}
			}
			to.Headers[key] = values[0]

			if strings.ToLower(key) == "content-type" {
				mimeType = values[0]
			}
		}
	}

	if res.Body != nil {
		if responseBody, readErr := ioutil.ReadAll(res.Body); readErr == nil {
			if strings.HasPrefix(mimeType, "text/") || strings.HasPrefix(mimeType, "application/json") || strings.HasPrefix(mimeType, "image/svg+xml") {
				to.IsBase64Encoded = false
				to.Body = string(responseBody)
			} else {
				to.IsBase64Encoded = true
				to.Body = base64.StdEncoding.EncodeToString(responseBody)
			}
		} else {
			err = readErr
		}
	}

	return
}

func FromAPIGatewayV2HTTPRequest2HttpRequest(from *events.APIGatewayV2HTTPRequest) (req *http.Request, err error) {
	req = &http.Request{
		Method: from.RequestContext.HTTP.Method,
	}

	if from.IsBase64Encoded {
		if decodedBody, decodeErr := base64.StdEncoding.DecodeString(from.Body); decodeErr == nil {
			req.Body = ioutil.NopCloser(bytes.NewReader(decodedBody))
			req.ContentLength = int64(len(decodedBody))
		} else {
			err = decodeErr
		}
	} else {
		req.Body = ioutil.NopCloser(bytes.NewReader([]byte(from.Body)))
		req.ContentLength = int64(len(from.Body))
	}

	if err == nil {
		if req.Header == nil {
			req.Header = http.Header{}
		}

		for key, value := range from.Headers {
			req.Header.Add(key, value)
		}

		baseURLbuilder := ThcompUtility.StringBuilder{}
		baseURLbuilder.Append("http://localhost/")
		if len(from.RequestContext.HTTP.Path) > 0 {
			tempPath := from.RequestContext.HTTP.Path
			tempPath = strings.TrimPrefix(tempPath, "/")
			baseURLbuilder.Append(tempPath)
		}

		if len(from.QueryStringParameters) > 0 {
			queries := []string{}
			for key, value := range from.QueryStringParameters {
				queries = append(queries, key+"="+value)
			}

			queriesText := strings.Join(queries, "&")
			baseURLbuilder.Append("?").Append(queriesText)
		}

		if tempURL, parseErr := url.Parse(baseURLbuilder.String()); parseErr == nil {
			req.URL = tempURL
		} else {
			err = parseErr
		}
	}

	return
}

func FromHttpResponse2APIGatewayV2HTTPResponse(res *http.Response) (to *events.APIGatewayV2HTTPResponse, err error) {
	to = &events.APIGatewayV2HTTPResponse{
		StatusCode: res.StatusCode,
	}

	mimeType := ``
	for key, values := range res.Header {
		if len(values) > 1 {
			if to.MultiValueHeaders == nil {
				to.MultiValueHeaders = map[string][]string{}
			}
			to.MultiValueHeaders[key] = values
		} else if len(values) == 1 {
			if to.Headers == nil {
				to.Headers = map[string]string{}
			}
			to.Headers[key] = values[0]

			if strings.ToLower(key) == "content-type" {
				mimeType = values[0]
			}
		}
	}

	if res.Body != nil {
		if responseBody, readErr := ioutil.ReadAll(res.Body); readErr == nil {
			if strings.HasPrefix(mimeType, "text/") || strings.HasPrefix(mimeType, "application/json") || strings.HasPrefix(mimeType, "image/svg+xml") {
				to.IsBase64Encoded = false
				to.Body = string(responseBody)
			} else {
				to.IsBase64Encoded = true
				to.Body = base64.StdEncoding.EncodeToString(responseBody)
			}
		} else {
			err = readErr
		}
	}

	return
}

func FromLambdaFunctionURLRequest2HttpRequest(from *events.LambdaFunctionURLRequest) (req *http.Request, err error) {
	req = &http.Request{
		Method: from.RequestContext.HTTP.Method,
	}

	if from.IsBase64Encoded {
		if decodedBody, decodeErr := base64.StdEncoding.DecodeString(from.Body); decodeErr == nil {
			req.Body = ioutil.NopCloser(bytes.NewReader(decodedBody))
			req.ContentLength = int64(len(decodedBody))
		} else {
			err = decodeErr
		}
	} else {
		req.Body = ioutil.NopCloser(bytes.NewReader([]byte(from.Body)))
		req.ContentLength = int64(len(from.Body))
	}

	if err == nil {
		if req.Header == nil {
			req.Header = http.Header{}
		}

		for key, value := range from.Headers {
			req.Header.Add(key, value)
		}

		baseURLbuilder := ThcompUtility.StringBuilder{}
		baseURLbuilder.Append("http://localhost/")
		if len(from.RequestContext.HTTP.Path) > 0 {
			tempPath := from.RequestContext.HTTP.Path
			tempPath = strings.TrimPrefix(tempPath, "/")
			baseURLbuilder.Append(tempPath)
		}

		if len(from.QueryStringParameters) > 0 {
			queries := []string{}
			for key, value := range from.QueryStringParameters {
				queries = append(queries, key+"="+value)
			}

			queriesText := strings.Join(queries, "&")
			baseURLbuilder.Append("?").Append(queriesText)
		}

		if tempURL, parseErr := url.Parse(baseURLbuilder.String()); parseErr == nil {
			req.URL = tempURL
		} else {
			err = parseErr
		}
	}

	return
}

func FromHttpResponse2LambdaFunctionURLResponse(res *http.Response) (to *events.LambdaFunctionURLResponse, err error) {
	to = &events.LambdaFunctionURLResponse{
		StatusCode: res.StatusCode,
	}

	mimeType := ``
	for key, values := range res.Header {
		if to.Headers == nil {
			to.Headers = map[string]string{}
		}
		to.Headers[key] = values[0]

		if strings.ToLower(key) == "content-type" {
			mimeType = values[0]
		}
	}

	if res.Body != nil {
		if responseBody, readErr := ioutil.ReadAll(res.Body); readErr == nil {
			if strings.HasPrefix(mimeType, "text/") || strings.HasPrefix(mimeType, "application/json") || strings.HasPrefix(mimeType, "image/svg+xml") {
				to.IsBase64Encoded = false
				to.Body = string(responseBody)
			} else {
				to.IsBase64Encoded = true
				to.Body = base64.StdEncoding.EncodeToString(responseBody)
			}
		} else {
			err = readErr
		}
	}

	return
}

func StartLambda(handler interface{}) {
	lambda.Start(handler)
}

func StartLambdaForHTTP(handler HttpRequestHandler) {
	info := httpHandlerInfo{
		userHttpRequestHandler: handler,
	}
	lambda.Start(info.Handler2)
}

func StartLambdaForApiGwProxy1(handler ApiGwProxyHandler1) {
	info := httpHandlerInfo{
		userApiGwProxyHandler1: handler,
	}
	lambda.Start(info.Handler1)
}

func StartLambdaForApiGwProxy2(handler ApiGwProxyHandler2) {
	info := httpHandlerInfo{
		userApiGwProxyHandler2: handler,
	}
	lambda.Start(info.Handler2)
}

func StartLambdaForApiGwV2Http1(handler ApiGwV2HttpHandler1) {
	info := httpHandlerInfo{
		userApiGwV2HttpHandler1: handler,
	}
	lambda.Start(info.Handler1)
}

func StartLambdaForApiGwV2Http2(handler ApiGwV2HttpHandler2) {
	info := httpHandlerInfo{
		userApiGwV2HttpHandler2: handler,
	}
	lambda.Start(info.Handler2)
}

func StartLambdaForApiGwWebsocket(handler ApiGwWebsocketHandler) {
	info := httpHandlerInfo{
		userApiGwWebsocketHandler: handler,
	}
	lambda.Start(info.Handler1)
}

func StartLambdaForLambdaFunctionURL1(handler LambdaFunctionURLHandler1) {
	info := httpHandlerInfo{
		userLambdaFunctionURLHandler1: handler,
	}
	lambda.Start(info.Handler1)
}

func StartLambdaForLambdaFunctionURL2(handler LambdaFunctionURLHandler2) {
	info := httpHandlerInfo{
		userLambdaFunctionURLHandler2: handler,
	}
	lambda.Start(info.Handler2)
}

func StartLambdaForSNS(handler SimpleNotificationServiceHandler) {
	info := httpHandlerInfo{
		userSimpleNotificationServiceHandler: handler,
	}
	lambda.Start(info.Handler1)
}

func StartLambdaForSQS(handler SimpleQueueServiceHandler) {
	info := httpHandlerInfo{
		userSimpleQueueServiceHandler: handler,
	}
	lambda.Start(info.Handler1)
}

func StartLambdaForSES(handler SimpleEmailEventHandler) {
	info := httpHandlerInfo{
		userSimpleEmailEventHandler: handler,
	}
	lambda.Start(info.Handler1)
}
