package publisher

import (
	"fmt"
	"testing"
)

func TestCreatePublisher(t *testing.T) {
	testCases := []string{"env", "volume"}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("tesing %s publisher", tc), func(t *testing.T) {
			publisher, err := CreatePublisher(tc)
			if err != nil {
				t.Errorf("Unexpected error creating publisher: %v", err)
			}

			var i interface{} = publisher
			_, ok := i.(Publisher)
			if !ok {
				t.Errorf("Returned publisher doesn't implement Publisher interface")
			}
		})
	}
}

func TestCreateUnknownPublisher(t *testing.T) {
	_, err := CreatePublisher("superpublisher")
	if err == nil {
		t.Error("No error when creating an unknown publisher")
	}
	_, ok := err.(*InvalidPublisherError)
	if !ok {
		t.Errorf("Unexpected error returned: %v", err)
	}
	expectedMsg := "Invalid publisher: superpublisher"
	if expectedMsg != err.Error() {
		t.Errorf("Received unexpected error message: %s", err.Error())
	}

}
