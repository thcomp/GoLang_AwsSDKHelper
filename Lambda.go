package awssdkhelper

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	ThcompUtility "github.com/thcomp/GoLang_Utility"
)

type HttpRequestHandler func(r *http.Request, w http.ResponseWriter)
type ApiGwProxyHandler1 func(event *events.APIGatewayProxyRequest) error
type ApiGwProxyHandler2 func(context *events.APIGatewayProxyRequestContext, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error)
type ApiGwV2HttpHandler1 func(event *events.APIGatewayV2HTTPRequest) error
type ApiGwV2HttpHandler2 func(context *events.APIGatewayV2HTTPRequestContext, event *events.APIGatewayV2HTTPRequest) (*events.APIGatewayV2HTTPResponse, error)
type ApiGwWebsocketHandler func(context *events.APIGatewayWebsocketProxyRequestContext, event *events.APIGatewayWebsocketProxyRequest) error
type LambdaFunctionURLHandler1 func(event *events.LambdaFunctionURLRequest) error
type LambdaFunctionURLHandler2 func(context *events.LambdaFunctionURLRequestContext, event *events.LambdaFunctionURLRequest) (*events.LambdaFunctionURLResponse, error)
type SimpleNotificationServiceHandler func(event *events.SNSEvent) error
type SimpleQueueServiceHandler func(event *events.SQSEvent) error
type SimpleEmailEventHandler func(event *events.SimpleEmailEvent) error

type LambdaEventType int

const (
	Unknown LambdaEventType = iota
	LambdaFunctionURL
	APIGateway
	APIGatewayV2
	APIGatewayWebsocket
	SNSEvent
	SQSEvent
	SimpleEmailEvent
	EventBridgeRules
	EventBridgeScheduler
)

type LambdaEventHelper struct {
	eventMap  map[string]interface{}
	eventType LambdaEventType
}

func NewLambdaEventHelper(event interface{}) (helper *LambdaEventHelper, err error) {
	if eventMap, assertionOK := event.(map[string]interface{}); assertionOK {
		if eventType, tempErr := wahtIsLambdaEvent(eventMap); eventType != Unknown && tempErr == nil {
			helper = &LambdaEventHelper{
				eventMap:  eventMap,
				eventType: eventType,
			}
		} else if tempErr != nil {
			err = tempErr
		} else {
			err = fmt.Errorf("unknown event type: %v", eventMap)
		}
	} else {
		err = fmt.Errorf("event is not a map[string]interface{}: %v: %s", event, reflect.TypeOf(event).String())
		return
	}

	return
}

func wahtIsLambdaEvent(event map[string]interface{}) (eventType LambdaEventType, err error) {
	eventType = Unknown

	if records, exist := event["Records"]; exist {
		if recordsArray, assertionOK := records.([]interface{}); assertionOK && len(recordsArray) > 0 {
			if recordMap, assertionOK := recordsArray[0].(map[string]interface{}); assertionOK {
				if _, exist := recordMap["Sns"]; exist {
					// SNSEvent
					eventType = SNSEvent
				} else if _, exist := recordMap["messageId"]; exist {
					// SQSEvent
					eventType = SQSEvent
				} else if _, exist := recordMap["ses"]; exist {
					// SimpleEmailEvent
					eventType = SimpleEmailEvent
				} else {
					err = fmt.Errorf("unknown record format: %v", recordMap)
				}
			} else {
				err = fmt.Errorf("records is not a map[string]interface{}: %v", recordsArray[0])
			}
		} else {
			err = fmt.Errorf("records is not an array of map[string]interface{}: %v", records)
		}
	} else if requestContext, exist := event["requestContext"]; exist {
		// APIGatewayProxyRequest, APIGatewayV2HTTPRequest, APIGatewayV2HTTPRequest or LambdaFunctionURLRequest
		if requestContextMap, assertionOK := requestContext.(map[string]interface{}); assertionOK {
			if _, exist := requestContextMap["status"]; exist {
				eventType = APIGatewayWebsocket
			} else if _, exist := requestContextMap["routeKey"]; exist {
				eventType = APIGatewayV2
			} else if _, exist := requestContextMap["httpMethod"]; exist {
				eventType = APIGateway
			} else if _, exist := requestContextMap["http"]; exist {
				eventType = LambdaFunctionURL
			} else {
				err = fmt.Errorf("unknown requestContext format: %v", requestContext)
			}
		}
	} else if source, exist := event["source"]; exist {
		switch source {
		case "aws.events":
			eventType = EventBridgeRules
		case "aws.scheduler":
			eventType = EventBridgeScheduler
		default:
			err = fmt.Errorf("unknown source: %s", source)
		}
	} else {
		err = fmt.Errorf("unknown event format: %v", event)
	}

	return
}

func (helper *LambdaEventHelper) HttpRequest() (req *http.Request, err error) {
	req = &http.Request{}

	// method
	if requestContext, exist := helper.eventMap["requestContext"]; exist {
		if requestContextMap, ok := requestContext.(map[string]interface{}); ok {
			var path, rawQuery string

			// Build URL
			urlStr := "http://localhost"
			if domainName, ok := requestContextMap["domainName"].(string); ok && domainName != "" {
				urlStr = "http://" + domainName
			}

			// LambdaFunctionURL, APIGatewayV2, APIGateway, Websocket
			if httpCtx, ok := requestContextMap["http"].(map[string]interface{}); ok {
				// LambdaFunctionURL or APIGatewayV2
				if m, ok := httpCtx["method"].(string); ok {
					req.Method = m
				}
				if p, ok := httpCtx["path"].(string); ok {
					path = p
				}
			} else {
				// APIGateway or Websocket
				if m, ok := helper.eventMap["httpMethod"].(string); ok {
					req.Method = m
				}
				if p, ok := helper.eventMap["path"].(string); ok {
					path = p
				}
			}

			if path != "" {
				if !strings.HasPrefix(path, "/") {
					urlStr += "/"
				}
				urlStr += path
			}

			// Try to get rawQueryString for v2/lambda-url
			if v, ok := helper.eventMap["rawQueryString"].(string); ok {
				rawQuery = v
			}

			// fallback: build query string from QueryStringParameters
			if rawQuery == "" {
				if qsp, ok := helper.eventMap["queryStringParameters"].(map[string]interface{}); ok && len(qsp) > 0 {
					queries := url.Values{}
					for k, v := range qsp {
						if s, ok := v.(string); ok {
							queries.Add(k, s)
						}
					}
					rawQuery = queries.Encode()
				}
			}

			// fallback: build query string from MultiValueQueryStringParameters
			if rawQuery == "" {
				if mvqsp, ok := helper.eventMap["multiValueQueryStringParameters"].(map[string]interface{}); ok && len(mvqsp) > 0 {
					queries := url.Values{}
					for k, v := range mvqsp {
						if arr, ok := v.([]interface{}); ok {
							for _, item := range arr {
								if s, ok := item.(string); ok {
									queries.Add(k, s)
								}
							}
						}
					}
					rawQuery = queries.Encode()
				}
			}

			if rawQuery != "" {
				urlStr += "?" + rawQuery
			}
			if u, err2 := url.Parse(urlStr); err2 == nil {
				req.URL = u
			} else {
				err = err2
			}
		}
	}

	if err == nil {
		// entity
		if body, exist := helper.eventMap["body"]; exist {
			isBase64Encoded := false
			if v, ok := helper.eventMap["isBase64Encoded"]; ok {
				if b, ok := v.(bool); ok {
					isBase64Encoded = b
				}
			}

			bodyBytes := []byte(nil)
			if strBody, ok := body.(string); ok {
				if isBase64Encoded {
					bodyBytes, err = base64.StdEncoding.DecodeString(strBody)
				} else {
					bodyBytes = []byte(strBody)
				}

				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				req.ContentLength = int64(len(bodyBytes))
			}
		}
	}

	if err == nil {
		// header
		if headers, exist := helper.eventMap["headers"]; exist {
			if req.Header == nil {
				req.Header = http.Header{}
			}

			if headersMap, assertionOK := headers.(map[string]interface{}); assertionOK {
				for key, value := range headersMap {
					if strVal, ok := value.(string); ok {
						req.Header.Add(key, strVal)
					} else if arrVal, ok := value.([]interface{}); ok {
						for _, v := range arrVal {
							if str, ok := v.(string); ok {
								req.Header.Add(key, str)
							}
						}
					}
				}
			} else if headersMap, assertionOK := headers.(map[string]string); assertionOK {
				for key, value := range headersMap {
					req.Header.Add(key, value)
				}
			} else if headersMap, assertionOK := headers.(map[string]([]string)); assertionOK {
				for key, values := range headersMap {
					for _, value := range values {
						req.Header.Add(key, value)
					}
				}
			}
		}

		// multiValueHeaders
		if multiValueHeaders, exist := helper.eventMap["multiValueHeaders"]; exist {
			if req.Header == nil {
				req.Header = http.Header{}
			}

			if mvhMap, ok := multiValueHeaders.(map[string]interface{}); ok {
				for key, value := range mvhMap {
					if arrVal, ok := value.([]interface{}); ok {
						for _, v := range arrVal {
							if str, ok := v.(string); ok {
								req.Header.Add(key, str)
							}
						}
					}
				}
			} else if mvhMap, ok := multiValueHeaders.(map[string][]string); ok {
				for key, values := range mvhMap {
					for _, value := range values {
						req.Header.Add(key, value)
					}
				}
			}
		}
	}

	return
}

func (helper *LambdaEventHelper) APIGatewayProxyRequest() (ret *events.APIGatewayProxyRequest, retErr error) {
	if helper.eventType == APIGateway {
		if jsonBytes, marshalErr := json.Marshal(helper.eventMap); marshalErr == nil {
			ret = &events.APIGatewayProxyRequest{}
			retErr = json.Unmarshal(jsonBytes, ret)
		} else {
			retErr = marshalErr
		}
	}

	return
}

func (helper *LambdaEventHelper) APIGatewayV2HTTPRequest() (ret *events.APIGatewayV2HTTPRequest, retErr error) {
	if helper.eventType == APIGatewayV2 {
		if jsonBytes, marshalErr := json.Marshal(helper.eventMap); marshalErr == nil {
			ret = &events.APIGatewayV2HTTPRequest{}
			retErr = json.Unmarshal(jsonBytes, ret)
		} else {
			retErr = marshalErr
		}
	}

	return
}

func (helper *LambdaEventHelper) LambdaFunctionURLRequest() (ret *events.LambdaFunctionURLRequest, retErr error) {
	if helper.eventType == LambdaFunctionURL {
		if jsonBytes, marshalErr := json.Marshal(helper.eventMap); marshalErr == nil {
			ret = &events.LambdaFunctionURLRequest{}
			retErr = json.Unmarshal(jsonBytes, ret)
		} else {
			retErr = marshalErr
		}
	}

	return
}

func (helper *LambdaEventHelper) SNSEvent() (ret *events.SNSEvent, retErr error) {
	if helper.eventType == SNSEvent {
		if jsonBytes, marshalErr := json.Marshal(helper.eventMap); marshalErr == nil {
			ret = &events.SNSEvent{}
			retErr = json.Unmarshal(jsonBytes, ret)
		} else {
			retErr = marshalErr
		}
	}

	return
}

func (helper *LambdaEventHelper) SQSEvent() (ret *events.SQSEvent, retErr error) {
	if helper.eventType == SQSEvent {
		if jsonBytes, marshalErr := json.Marshal(helper.eventMap); marshalErr == nil {
			ret = &events.SQSEvent{}
			retErr = json.Unmarshal(jsonBytes, ret)
		} else {
			retErr = marshalErr
		}
	}

	return
}

func (helper *LambdaEventHelper) SimpleEmailEvent() (ret *events.SimpleEmailEvent, retErr error) {
	if helper.eventType == SimpleEmailEvent {
		if jsonBytes, marshalErr := json.Marshal(helper.eventMap); marshalErr == nil {
			ret = &events.SimpleEmailEvent{}
			retErr = json.Unmarshal(jsonBytes, ret)
		} else {
			retErr = marshalErr
		}
	}

	return
}

func (helper *LambdaEventHelper) EventBridgeEvent() (ret *events.EventBridgeEvent, retErr error) {
	switch helper.eventType {
	case EventBridgeRules, EventBridgeScheduler:
		if jsonBytes, marshalErr := json.Marshal(helper.eventMap); marshalErr == nil {
			ret = &events.EventBridgeEvent{}
			retErr = json.Unmarshal(jsonBytes, ret)
		} else {
			retErr = marshalErr
		}
	}

	return
}

func (helper *LambdaEventHelper) Method() (method string, err error) {
	if requestContext, exist := helper.eventMap["requestContext"]; exist {
		if requestContextMap, ok := requestContext.(map[string]interface{}); ok {
			// LambdaFunctionURL or APIGatewayV2
			if httpCtx, ok := requestContextMap["http"].(map[string]interface{}); ok {
				if m, ok := httpCtx["method"].(string); ok {
					method = m
				}
			}
			// APIGateway or Websocket
			if m, ok := helper.eventMap["httpMethod"].(string); ok {
				method = m
			}
		}
	} else {
		err = fmt.Errorf("method not found in eventMap")
	}

	return
}

func (helper *LambdaEventHelper) Headers() (headers http.Header, err error) {
	headers = http.Header{}

	// headers
	if h, exist := helper.eventMap["headers"]; exist {
		switch v := h.(type) {
		case map[string]interface{}:
			for key, value := range v {
				if strVal, ok := value.(string); ok {
					headers.Add(key, strVal)
				} else if arrVal, ok := value.([]interface{}); ok {
					for _, item := range arrVal {
						if str, ok := item.(string); ok {
							headers.Add(key, str)
						}
					}
				}
			}
		case map[string]string:
			for key, value := range v {
				headers.Add(key, value)
			}
		case map[string][]string:
			for key, values := range v {
				for _, value := range values {
					headers.Add(key, value)
				}
			}
		}
	}

	// multiValueHeaders
	if mvh, exist := helper.eventMap["multiValueHeaders"]; exist {
		switch v := mvh.(type) {
		case map[string]interface{}:
			for key, value := range v {
				if arrVal, ok := value.([]interface{}); ok {
					for _, item := range arrVal {
						if str, ok := item.(string); ok {
							headers.Add(key, str)
						}
					}
				}
			}
		case map[string][]string:
			for key, values := range v {
				for _, value := range values {
					headers.Add(key, value)
				}
			}
		}
	}

	return
}

func (helper *LambdaEventHelper) Body() (body io.ReadCloser, err error) {
	if rawBody, exist := helper.eventMap["body"]; exist {
		isBase64Encoded := false
		if v, ok := helper.eventMap["isBase64Encoded"]; ok {
			if b, ok := v.(bool); ok {
				isBase64Encoded = b
			}
		}

		if strBody, ok := rawBody.(string); ok {
			var bodyBytes []byte
			if isBase64Encoded {
				bodyBytes, err = base64.StdEncoding.DecodeString(strBody)
			} else {
				bodyBytes = []byte(strBody)
			}
			if err == nil {
				body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
		} else {
			err = fmt.Errorf("body is not a string: %v", rawBody)
		}
	} else {
		err = fmt.Errorf("body is not in eventMap")
	}

	return
}

func (helper *LambdaEventHelper) URL() (u *url.URL, err error) {
	var urlStr, path, rawQuery string

	urlStr = "http://localhost"

	if requestContext, exist := helper.eventMap["requestContext"]; exist {
		if requestContextMap, ok := requestContext.(map[string]interface{}); ok {
			if domainName, ok := requestContextMap["domainName"].(string); ok && domainName != "" {
				urlStr = "http://" + domainName
			}
			// LambdaFunctionURL, APIGatewayV2, APIGateway, Websocket
			if httpCtx, ok := requestContextMap["http"].(map[string]interface{}); ok {
				if p, ok := httpCtx["path"].(string); ok {
					path = p
				}
			} else {
				if p, ok := helper.eventMap["path"].(string); ok {
					path = p
				}
			}
		}
	}

	if path != "" {
		if !strings.HasPrefix(path, "/") {
			urlStr += "/"
		}
		urlStr += path
	}

	// Try to get rawQueryString for v2/lambda-url
	if v, ok := helper.eventMap["rawQueryString"].(string); ok && v != "" {
		rawQuery = v
	}

	// fallback: build query string from QueryStringParameters
	if rawQuery == "" {
		if qsp, ok := helper.eventMap["queryStringParameters"].(map[string]interface{}); ok && len(qsp) > 0 {
			queries := url.Values{}
			for k, v := range qsp {
				if s, ok := v.(string); ok {
					queries.Add(k, s)
				}
			}
			rawQuery = queries.Encode()
		}
	}

	// fallback: build query string from MultiValueQueryStringParameters
	if rawQuery == "" {
		if mvqsp, ok := helper.eventMap["multiValueQueryStringParameters"].(map[string]interface{}); ok && len(mvqsp) > 0 {
			queries := url.Values{}
			for k, v := range mvqsp {
				if arr, ok := v.([]interface{}); ok {
					for _, item := range arr {
						if s, ok := item.(string); ok {
							queries.Add(k, s)
						}
					}
				}
			}
			rawQuery = queries.Encode()
		}
	}

	if rawQuery != "" {
		urlStr += "?" + rawQuery
	}

	u, err = url.Parse(urlStr)
	return
}

func (helper *LambdaEventHelper) MapOfAPIGatewayProxyResponse(response *http.Response) (ret map[string]interface{}, retErr error) {
	headers := make(map[string]string)
	multiValueHeaders := make(map[string][]string)

	for k, v := range response.Header {
		headers[k] = v[0]        // Take the first value for single-value headers
		multiValueHeaders[k] = v // Keep all values for multi-value headers
	}

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	isBase64Encoded := false
	contentType := response.Header.Get("Content-Type")
	// Content-TypeからMIMEタイプをパース
	if mimeType, _, err := mime.ParseMediaType(contentType); err == nil {
		if !strings.HasPrefix(mimeType, "text/") && !strings.Contains(mimeType, "json") && !strings.Contains(mimeType, "xml") {
			// "text/"で始まらない、かつ "json" も "xml" も含まれない場合はバイナリとみなす
			// JSONやXMLは通常テキストなので、明示的に除外
			isBase64Encoded = true
		}
	} else {
		isBase64Encoded = true
	}

	var bodyString string
	if isBase64Encoded {
		bodyString = base64.StdEncoding.EncodeToString(bodyBytes)
	} else {
		bodyString = string(bodyBytes)
	}

	responseMap := map[string]interface{}{
		"statusCode":        response.StatusCode,
		"headers":           headers,
		"multiValueHeaders": multiValueHeaders, // Optionnal: Use if you need multi-value headers
		"body":              bodyString,
		"isBase64Encoded":   isBase64Encoded,
	}

	return responseMap, nil
}

func (helper *LambdaEventHelper) MapOfAPIGatewayV2HTTPResponse(response *http.Response) (ret map[string]interface{}, retErr error) {
	headers := make(map[string]string)
	multiValueHeaders := make(map[string][]string)
	var cookies []string

	for k, v := range response.Header {
		lowerKey := strings.ToLower(k)
		if lowerKey == "set-cookie" {
			cookies = append(cookies, v...) // Set-Cookie は複数ある可能性があるので全て追加
		} else {
			headers[k] = v[0]        // カンマ区切りで結合
			multiValueHeaders[k] = v // Keep all values for multi-value headers
		}
	}

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close() // Bodyを読み終わったらクローズする

	isBase64Encoded := false
	contentType := response.Header.Get("Content-Type")
	// Content-TypeからMIMEタイプをパース
	if mimeType, _, err := mime.ParseMediaType(contentType); err == nil {
		if !strings.HasPrefix(mimeType, "text/") && !strings.Contains(mimeType, "json") && !strings.Contains(mimeType, "xml") {
			// "text/"で始まらない、かつ "json" も "xml" も含まれない場合はバイナリとみなす
			// JSONやXMLは通常テキストなので、明示的に除外
			isBase64Encoded = true
		}
	} else {
		isBase64Encoded = true
	}

	var bodyString string
	if isBase64Encoded {
		bodyString = base64.StdEncoding.EncodeToString(bodyBytes)
	} else {
		bodyString = string(bodyBytes)
	}

	responseMap := map[string]interface{}{
		"statusCode":        response.StatusCode,
		"headers":           headers,
		"multiValueHeaders": multiValueHeaders,
		"body":              bodyString,
		"isBase64Encoded":   isBase64Encoded,
		"cookies":           cookies, // APIGatewayV2HTTPResponse の Cookies フィールド
	}

	return responseMap, nil
}

func (helper *LambdaEventHelper) MapOfLambdaFunctionURLResponse(response *http.Response) (ret map[string]interface{}, retErr error) {
	headers := make(map[string]string)

	for k, v := range response.Header {
		// Lambda Function URL Response の Headers は map[string]string なので、
		// 複数値のヘッダーはカンマ区切りで結合します。
		// 例えば、Set-Cookie は複数の値を持つことがありますが、Lambda Function URL では
		// これらを単一の "Set-Cookie" ヘッダーとしてカンマ区切りで結合するのが一般的です。
		// ただし、RFC 6265 に基づくと Set-Cookie は個別のヘッダーとして送るべきですが、
		// Lambda Function URL の制限によりこのように処理します。
		headers[k] = strings.Join(v, ", ")
	}

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close() // Bodyを読み終わったらクローズする

	contentType := response.Header.Get("Content-Type")
	mimeType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		// パースエラーが発生した場合は、元のContent-TypeをそのままMIMEタイプとして使う
		mimeType = contentType
	}

	isBase64Encoded := false
	// text/で始まる、またはapplication/json, application/xml, application/javascript, text/css
	// の場合はテキストとみなす。
	if !strings.HasPrefix(mimeType, "text/") &&
		!strings.Contains(mimeType, "json") &&
		!strings.Contains(mimeType, "xml") &&
		!strings.Contains(mimeType, "javascript") &&
		!strings.Contains(mimeType, "css") {
		isBase64Encoded = true
	}

	var bodyString string
	if isBase64Encoded {
		bodyString = base64.StdEncoding.EncodeToString(bodyBytes)
	} else {
		bodyString = string(bodyBytes)
	}

	responseMap := map[string]interface{}{
		"statusCode":      response.StatusCode,
		"headers":         headers,
		"body":            bodyString,
		"isBase64Encoded": isBase64Encoded,
	}

	return responseMap, nil
}

func FromAPIGatewayProxyRequest2HttpRequest(from *events.APIGatewayProxyRequest) (req *http.Request, err error) {
	req = &http.Request{
		Method: from.HTTPMethod,
	}

	if from.IsBase64Encoded {
		if decodedBody, decodeErr := base64.StdEncoding.DecodeString(from.Body); decodeErr == nil {
			req.Body = io.NopCloser(bytes.NewReader(decodedBody))
			req.ContentLength = int64(len(decodedBody))
		} else {
			err = decodeErr
		}
	} else {
		req.Body = io.NopCloser(bytes.NewReader([]byte(from.Body)))
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
		if responseBody, readErr := io.ReadAll(res.Body); readErr == nil {
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
			req.Body = io.NopCloser(bytes.NewReader(decodedBody))
			req.ContentLength = int64(len(decodedBody))
		} else {
			err = decodeErr
		}
	} else {
		req.Body = io.NopCloser(bytes.NewReader([]byte(from.Body)))
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
		if responseBody, readErr := io.ReadAll(res.Body); readErr == nil {
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
			req.Body = io.NopCloser(bytes.NewReader(decodedBody))
			req.ContentLength = int64(len(decodedBody))
		} else {
			err = decodeErr
		}
	} else {
		req.Body = io.NopCloser(bytes.NewReader([]byte(from.Body)))
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
		if responseBody, readErr := io.ReadAll(res.Body); readErr == nil {
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

func StartLambda1(handler func(context context.Context, event interface{}) (err error)) {
	lambda.Start(handler)
}

func StartLambda2(handler func(context context.Context, event interface{}) (out interface{}, err error)) {
	lambda.Start(handler)
}

func StartLambdaForSNS(handler SimpleNotificationServiceHandler) {
	lambda.Start(handler)
}

func StartLambdaForSQS(handler SimpleQueueServiceHandler) {
	lambda.Start(handler)
}

func StartLambdaForSES(handler SimpleEmailEventHandler) {
	lambda.Start(handler)
}

func IsRunOnLambda() (ret bool) {
	serverlessPlatform := os.Getenv("serverless_platform")
	serverlessPlatform = strings.ToLower(serverlessPlatform)

	switch serverlessPlatform {
	case "lambda":
		ret = true
	case "aws_lambda", "aws lambda", "aws-lambda":
		ret = true
	}

	return
}
