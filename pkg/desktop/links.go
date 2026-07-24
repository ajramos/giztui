package desktop

import "context"

// ListLinks returns the links found in a message body.
func (a *API) ListLinks(ctx context.Context, id string) ([]Link, error) {
	if a.link == nil {
		return []Link{}, nil
	}
	infos, err := a.link.GetMessageLinks(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Link, 0, len(infos))
	for _, l := range infos {
		out = append(out, Link{Index: l.Index, URL: l.URL, Text: l.Text, Type: l.Type})
	}
	return out, nil
}
