package awssdkhelper

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"testing"

	TestUtility "github.com/thcomp/GoLang_TestUtility"
)

func TestLambdaEventHelper1(t *testing.T) {
	tester := TestUtility.NewTestHelper(t)
	testJson := `
		{
		"body": "eyJ0ZXN0IjoiYm9keSJ9",
		"resource": "/{proxy+}",
		"path": "/path/to/resource",
		"httpMethod": "POST",
		"isBase64Encoded": true,
		"queryStringParameters": {
			"foo": "bar",
			"foo2": "bar1",
			"foo2": "bar2",
		},
		"multiValueQueryStringParameters": {
			"foo2": [
			"bar1","bar2"
			]
		},
		"pathParameters": {
			"proxy": "/path/to/resource"
		},
		"stageVariables": {
			"baz": "qux"
		},
		"headers": {
			"Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
			"Accept-Encoding": "gzip, deflate, sdch",
			"Accept-Language": "en-US,en;q=0.8",
			"Cache-Control": "max-age=0",
			"CloudFront-Forwarded-Proto": "https",
			"CloudFront-Is-Desktop-Viewer": "true",
			"CloudFront-Is-Mobile-Viewer": "false",
			"CloudFront-Is-SmartTV-Viewer": "false",
			"CloudFront-Is-Tablet-Viewer": "false",
			"CloudFront-Viewer-Country": "US",
			"Host": "1234567890.execute-api.us-east-1.amazonaws.com",
			"Upgrade-Insecure-Requests": "1",
			"User-Agent": "Custom User Agent String",
			"Via": "1.1 08f323deadbeefa7af34d5feb414ce27.cloudfront.net (CloudFront)",
			"Via": "1.2 08f323deadbeefa7af34d5feb414ce27.cloudfront.net (CloudFront)",
			"X-Amz-Cf-Id": "cDehVQoZnx43VYQb9j2-nvCh-9z396Uhbp027Y2JvkCPNLmGJHqlaA==",
			"X-Forwarded-For": "127.0.0.1, 127.0.0.2",
			"X-Forwarded-Port": "443",
			"X-Forwarded-Proto": "https"
		},
		"multiValueHeaders": {
			"Accept": [
			"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8"
			],
			"Accept-Encoding": [
			"gzip, deflate, sdch"
			],
			"Accept-Language": [
			"en-US,en;q=0.8"
			],
			"Cache-Control": [
			"max-age=0"
			],
			"CloudFront-Forwarded-Proto": [
			"https"
			],
			"CloudFront-Is-Desktop-Viewer": [
			"true"
			],
			"CloudFront-Is-Mobile-Viewer": [
			"false"
			],
			"CloudFront-Is-SmartTV-Viewer": [
			"false"
			],
			"CloudFront-Is-Tablet-Viewer": [
			"false"
			],
			"CloudFront-Viewer-Country": [
			"US"
			],
			"Host": [
			"0123456789.execute-api.us-east-1.amazonaws.com"
			],
			"Upgrade-Insecure-Requests": [
			"1"
			],
			"User-Agent": [
			"Custom User Agent String"
			],
			"Via": [
			"1.1 08f323deadbeefa7af34d5feb414ce27.cloudfront.net (CloudFront)",
			"1.2 08f323deadbeefa7af34d5feb414ce27.cloudfront.net (CloudFront)"
			],
			"X-Amz-Cf-Id": [
			"cDehVQoZnx43VYQb9j2-nvCh-9z396Uhbp027Y2JvkCPNLmGJHqlaA=="
			],
			"X-Forwarded-For": [
			"127.0.0.1, 127.0.0.2"
			],
			"X-Forwarded-Port": [
			"443"
			],
			"X-Forwarded-Proto": [
			"https"
			]
		},
		"requestContext": {
			"accountId": "123456789012",
			"resourceId": "123456",
			"stage": "prod",
			"requestId": "c6af9ac6-7b61-11e6-9a41-93e8deadbeef",
			"requestTime": "09/Apr/2015:12:34:56 +0000",
			"requestTimeEpoch": 1428582896000,
			"identity": {
			"cognitoIdentityPoolId": null,
			"accountId": null,
			"cognitoIdentityId": null,
			"caller": null,
			"accessKey": null,
			"sourceIp": "127.0.0.1",
			"cognitoAuthenticationType": null,
			"cognitoAuthenticationProvider": null,
			"userArn": null,
			"userAgent": "Custom User Agent String",
			"user": null
			},
			"path": "/prod/path/to/resource",
			"resourcePath": "/{proxy+}",
			"httpMethod": "POST",
			"apiId": "1234567890",
			"protocol": "HTTP/1.1"
		}
	}	
	`
	eventMap := map[string]interface{}{}
	if unmarshalErr := json.Unmarshal([]byte(testJson), &eventMap); unmarshalErr == nil {
		if helper, err := NewLambdaEventHelper(eventMap); err == nil {
			tester.Errorf2(
				func() bool {
					if method, _ := helper.Method(); method == "POST" {
						return true
					} else {
						return false
					}
				}, "method is not POST",
			)

			url, _ := helper.URL()
			tester.Errorf2(
				func() bool {
					if url.Host == "localhost" {
						return true
					} else {
						return false
					}
				}, "host is not localhost: %s", url.Host,
			)
			tester.Errorf2(
				func() bool {
					if url.Path == "/path/to/resource" {
						return true
					} else {
						return false
					}
				}, "path is not /path/to/resource",
			)
			tester.Errorf2(
				func() bool {
					if url.Query().Get("foo") == "bar" {
						return true
					} else {
						return false
					}
				}, "query foo is not bar: %s", url.Query().Get("foo"),
			)

			foo2Value := []string(nil)
			tester.Errorf2(
				func() bool {
					for key, values := range url.Query() {
						if key == "foo2" {
							bar1Exist, bar2Exist, unknownExist := false, false, false
							foo2Value = values
							for _, value := range values {
								if value == "bar1" {
									bar1Exist = true
								} else if value == "bar2" {
									bar2Exist = true
								} else {
									unknownExist = true
								}
							}

							return bar1Exist && bar2Exist && !unknownExist
						}
					}

					return false
				}, "multi value query foo2 is not bar1 and bar2: %v", foo2Value,
			)

			headers, _ := helper.Headers()
			checkerUpgradeInsecureRequest, checkerVia := false, false
			tester.Errorf2(
				func() bool {
					for key, values := range headers {
						if key == "Upgrade-Insecure-Requests" && len(values) == 1 {
							checkerUpgradeInsecureRequest = true
						} else if key == "Via" && len(values) == 2 {
							checkerVia = true
						}
					}

					return checkerUpgradeInsecureRequest && checkerVia
				}, "header is not matched: %t, %t", checkerUpgradeInsecureRequest, checkerVia,
			)
			entity, _ := helper.Body()
			data, _ := io.ReadAll(entity)
			origin, _ := base64.StdEncoding.DecodeString("eyJ0ZXN0IjoiYm9keSJ9")
			tester.Errorf(string(data) == string(origin), "body not matched to %s: %s", string(origin), string(data))
		}
	}
}
