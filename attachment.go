package confluence

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
)

// https://docs.atlassian.com/atlassian-confluence/REST/6.5.2/#content/{id}/child/attachment

type Attachments struct {
	Results []Attachment `json:"results"`
	Size    int          `json:"size"`
}

type Attachment struct {
	Id       string `json:"id"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Title    string `json:"title"`
	Metadata struct {
		Comment   string `json:"comment"`
		MediaType string `json:"mediaType"`
	} `json:"metadata"`
	Version struct {
		Number int `json:"number"`
	} `json:"version"`
}

func (client *Client) newAttachmentEndpoint(contentID string) (*url.URL, error) {
	return url.ParseRequestURI(client.Endpoint + "/content/" + contentID + "/child/attachment")
}

func (client *Client) attachmentEndpoint(contentID, attachmentID string) (*url.URL, error) {
	if endpoint, err := client.newAttachmentEndpoint(contentID); err == nil {
		return url.ParseRequestURI(endpoint.String() + "/" + attachmentID)
	} else {
		return nil, err
	}
}

func (client *Client) attachmentDataEndpoint(contentID, attachmentID string) (*url.URL, error) {
	if endpoint, err := client.attachmentEndpoint(contentID, attachmentID); err == nil {
		return url.ParseRequestURI(endpoint.String() + "/data")
	} else {
		return nil, err
	}
}

// DeleteAttachment ..
func (client *Client) DeleteAttachment(contentID string, attachmentID string) error {
	endpoint, err := client.attachmentEndpoint(contentID, attachmentID)
	if err != nil {
		return err
	}

	_, err = client.request("DELETE", endpoint.String(), "", "")
	if err != nil {
		return err
	}

	return nil
}

// GetAttachment ...
func (client *Client) GetAttachment(contentID, attachmentID string) (*Attachment, error) {
	endpoint, err := client.attachmentEndpoint(contentID, attachmentID)
	if err != nil {
		return nil, err
	}

	res, err := client.request("GET", endpoint.String(), "", "")
	if err != nil {
		return nil, err
	}

	var attachments Attachments
	err = json.Unmarshal(res, &attachments)
	if err != nil {
		return nil, err
	}
	if len(attachments.Results) < 1 {
		return nil, fmt.Errorf("empty list")
	}

	return &attachments.Results[0], nil
}

// GetAttachmentByFilename ...
func (client *Client) GetAttachmentByFilename(contentID, filename string) (*Attachment, error) {
	endpoint, err := client.newAttachmentEndpoint(contentID)
	if err != nil {
		return nil, err
	}
	data := url.Values{}
	data.Set("filename", filename)
	endpoint.RawQuery = data.Encode()

	res, err := client.request("GET", endpoint.String(), "", "")
	if err != nil {
		return nil, err
	}

	var attachments Attachments
	err = json.Unmarshal(res, &attachments)
	if err != nil {
		return nil, err
	}
	if len(attachments.Results) < 1 {
		return nil, fmt.Errorf("empty list")
	}

	return &attachments.Results[0], nil
}

// UpdateAttachment ...
func (client *Client) UpdateAttachment(contentID, attachmentID, path string, minorEdit bool) (*Attachment, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}

	part, err := writer.CreateFormFile("file", fi.Name())
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}

	err = writer.WriteField("minorEdit", strconv.FormatBool(minorEdit))
	if err != nil {
		return nil, err
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	endpoint, err := client.attachmentDataEndpoint(contentID, attachmentID)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", endpoint.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := client.sendRequest(req)
	if err != nil {
		return nil, err
	}

	var attachment Attachment
	err = json.Unmarshal(res, &attachment)
	if err != nil {
		return nil, err
	}
	return &attachment, nil
}

// AddAttachment ...
func (client *Client) AddAttachment(contentID, path string) (*Attachment, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}

	part, err := writer.CreateFormFile("file", fi.Name())
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}
	endpoint, err := client.newAttachmentEndpoint(contentID)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", endpoint.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := client.sendRequest(req)
	if err != nil {
		return nil, err
	}

	var attachments Attachments
	err = json.Unmarshal(res, &attachments)
	if err != nil {
		return nil, err
	}
	if len(attachments.Results) < 1 {
		return nil, fmt.Errorf("empty list")
	}

	return &attachments.Results[0], nil
}

// AddUpdateAttachments ...
func (client *Client) AddUpdateAttachments(contentID string, files []string) ([]*Attachment, []error) {
	var results []*Attachment
	var errors []error
	for _, f := range files {
		filename := path.Base(f)
		attachment, err := client.GetAttachmentByFilename(contentID, filename)
		if err != nil {
			attachment, err = client.AddAttachment(contentID, f)
		} else {
			attachment, err = client.UpdateAttachment(contentID, attachment.Id[3:], f, true)
		}
		if err == nil {
			results = append(results, attachment)
		} else {
			errors = append(errors, err)
		}
	}
	return results, errors
}
