package request

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sync"

	exception "github.com/blendlabs/go-exception"
)

// MockedResponse is the metadata and response body for a response
type MockedResponse struct {
	ResponseBody []byte
	StatusCode   int
	Error        error
}

// MockedResponseGenerator is a function that returns a mocked response.
type MockedResponseGenerator func() MockedResponse

var (
	isMocked  bool
	mocksLock sync.Mutex
	mocks     map[string]MockedResponseGenerator
)

// MockedResponseInjector injects the mocked response into the request response.
func MockedResponseInjector(verb string, workingURL *url.URL) (bool, *ResponseMeta, []byte, error) {
	if isMocked {
		mocksLock.Lock()
		storedURL := fmt.Sprintf("%s_%s", verb, workingURL.String())
		mockResponseHandler, ok := mocks[storedURL]
		mocksLock.Unlock()
		if ok {
			mockResponse := mockResponseHandler()
			meta := &ResponseMeta{}
			meta.StatusCode = mockResponse.StatusCode
			meta.ContentLength = int64(len(mockResponse.ResponseBody))
			return true, meta, mockResponse.ResponseBody, mockResponse.Error
		}
		panic(fmt.Sprintf("attempted to make service request w/o mocking endpoint: %s %s", verb, workingURL.String()))
	} else {
		return false, nil, nil, nil
	}
}

// MockResponse mocks are response with a given generator.
func MockResponse(verb string, url string, gen MockedResponseGenerator) {
	mocksLock.Lock()
	defer mocksLock.Unlock()

	MockResponseUnsafe(verb, url, gen)
}

// MockResponseUnsafe mocks are response with a given generator without locking.
func MockResponseUnsafe(verb string, url string, gen MockedResponseGenerator) {
	isMocked = true
	if mocks == nil {
		mocks = map[string]MockedResponseGenerator{}
	}
	storedURL := fmt.Sprintf("%s_%s", verb, url)
	mocks[storedURL] = gen
}

// MockResponseFromBinary mocks a service request response from a set of binary responses.
func MockResponseFromBinary(verb string, url string, statusCode int, responseBody []byte) {
	MockResponse(verb, url, func() MockedResponse {
		return MockedResponse{
			StatusCode:   statusCode,
			ResponseBody: responseBody,
		}
	})
}

// MockResponseFromBinaryUnsafe mocks a service request response from a set of binary responses without locking.
func MockResponseFromBinaryUnsafe(verb string, url string, statusCode int, responseBody []byte) {
	MockResponseUnsafe(verb, url, func() MockedResponse {
		return MockedResponse{
			StatusCode:   statusCode,
			ResponseBody: responseBody,
		}
	})
}

// MockResponseFromString mocks a service request response from a string responseBody.
func MockResponseFromString(verb string, url string, statusCode int, responseBody string) {
	MockResponseFromBinary(verb, url, statusCode, []byte(responseBody))
}

// MockResponseFromStringUnsafe mocks a service request response from a string responseBody without locking.
func MockResponseFromStringUnsafe(verb string, url string, statusCode int, responseBody string) {
	MockResponseFromBinaryUnsafe(verb, url, statusCode, []byte(responseBody))
}

// MockResponseFromFile mocks a service request response from a set of file paths.
func MockResponseFromFile(verb string, url string, statusCode int, responseFilePath string) {
	MockResponse(verb, url, readFile(statusCode, responseFilePath))
}

// MockResponseFromFileUnsafe mocks a service request response from a set of file paths without locking.
func MockResponseFromFileUnsafe(verb string, url string, statusCode int, responseFilePath string) {
	MockResponseUnsafe(verb, url, readFile(statusCode, responseFilePath))
}

func readFile(statusCode int, filePath string) func() MockedResponse {
	return func() MockedResponse {
		f, err := os.Open(filePath)
		if err != nil {
			return MockedResponse{
				StatusCode: statusCode,
				Error:      err,
			}
		}
		defer f.Close()

		contents, err := ioutil.ReadAll(f)
		if err != nil {
			return MockedResponse{
				StatusCode: statusCode,
				Error:      err,
			}
		}

		return MockedResponse{
			StatusCode:   statusCode,
			ResponseBody: contents,
		}
	}
}

// MockError mocks a service request error.
func MockError(verb string, url string) {
	MockResponse(verb, url, func() MockedResponse {
		return MockedResponse{
			StatusCode: http.StatusInternalServerError,
			Error:      exception.New("Error! This is from request#MockError. If you don't want an error don't mock it."),
		}
	})
}

// MockErrorUnsafe mocks a service request error without locking.
func MockErrorUnsafe(verb string, url string) {
	MockResponseUnsafe(verb, url, func() MockedResponse {
		return MockedResponse{
			StatusCode: http.StatusInternalServerError,
			Error:      exception.New("Error! This is from request#MockError. If you don't want an error don't mock it."),
		}
	})
}

// ClearMockedResponses clears any mocked responses that have been set up for the test.
func ClearMockedResponses() {
	mocksLock.Lock()
	defer mocksLock.Unlock()

	isMocked = false
	mocks = map[string]MockedResponseGenerator{}
}
