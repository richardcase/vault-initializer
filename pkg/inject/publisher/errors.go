package publisher

import "fmt"

func NewInvalidPublisherError(publisher string) *InvalidPublisherError {
	return &InvalidPublisherError{
		Publisher: publisher,
	}
}

type InvalidPublisherError struct {
	Publisher string
}

func (e *InvalidPublisherError) Error() string {
	return fmt.Sprintf("Invalid publisher: %s", e.Publisher)
}
