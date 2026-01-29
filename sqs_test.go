package awssdkhelper

import (
	"encoding/json"
	"os"
	"testing"
)

func Test_SQSHelper(t *testing.T) {
	if reader, openErr := os.Open("test_sqs.json"); openErr == nil {
		paramMap := map[string]string{}
		json.NewDecoder(reader).Decode(&paramMap)

		accessKeyId, _ := paramMap["access_key_id"]
		secretAccessKey, _ := paramMap["secret_access_key"]
		queueURL, _ := paramMap["queue_url"]
		region, _ := paramMap["region"]

		helper := NewSQSHelperWithIAM(queueURL, region, accessKeyId, secretAccessKey)
		if id, seqNum, sendErr := helper.SendMessage("test"); sendErr == nil {
			t.Logf("data: %s, %s\n", id, seqNum)
		} else {
			t.Fatalf("SendMessage error: %v", sendErr)
		}
	} else {
		t.Fatalf("open error: %v", openErr)
	}

}
