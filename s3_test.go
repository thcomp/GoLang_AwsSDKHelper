package awssdkhelper

import (
	"encoding/json"
	"io"
	"os"
	"testing"
)

func Test_S3Helper_GetItem(t *testing.T) {
	if reader, openErr := os.Open("test_s3.json"); openErr == nil {
		paramMap := map[string]string{}
		json.NewDecoder(reader).Decode(&paramMap)

		accessKeyId, _ := paramMap["access_key_id"]
		secretAccessKey, _ := paramMap["secret_access_key"]
		bucket, _ := paramMap["bucket"]
		region, _ := paramMap["region"]
		filepath, _ := paramMap["ipv4_address_filepath"]

		helper := NewS3Helper(accessKeyId, secretAccessKey, region, bucket, nil)
		if s3Item, getErr := helper.GetItem(filepath); getErr == nil {
			defer s3Item.Close()

			if data, readErr := io.ReadAll(s3Item); readErr == nil {
				t.Logf("data: %s\n", string(data))
			} else {
				t.Fatalf("ReadAll error: %v", readErr)
			}
		} else {
			t.Fatalf("GetItem error: %v", getErr)
		}
	} else {
		t.Fatalf("open error: %v", openErr)
	}

}
