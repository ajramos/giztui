package desktop

import (
	"context"
	"fmt"
)

// ListAttachments returns the attachments of a message (empty when none or when
// the attachment service is unavailable).
func (a *API) ListAttachments(ctx context.Context, id string) ([]Attachment, error) {
	if a.attach == nil {
		return []Attachment{}, nil
	}
	infos, err := a.attach.GetMessageAttachments(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Attachment, 0, len(infos))
	for _, in := range infos {
		out = append(out, Attachment{
			AttachmentID: in.AttachmentID,
			Filename:     in.Filename,
			MimeType:     in.MimeType,
			Size:         in.Size,
			Type:         in.Type,
			Inline:       in.Inline,
		})
	}
	return out, nil
}

// DownloadAttachment saves an attachment to the configured download directory
// and returns the final path on disk.
func (a *API) DownloadAttachment(ctx context.Context, messageID, attachmentID, filename string) (string, error) {
	if a.attach == nil {
		return "", fmt.Errorf("attachment service not available")
	}
	// Empty savePath => default download dir + suggested filename.
	return a.attach.DownloadAttachmentWithFilename(ctx, messageID, attachmentID, "", filename)
}

// OpenAttachment opens a previously-downloaded file with the OS default app.
func (a *API) OpenAttachment(ctx context.Context, path string) error {
	if a.attach == nil {
		return fmt.Errorf("attachment service not available")
	}
	return a.attach.OpenAttachment(ctx, path)
}
