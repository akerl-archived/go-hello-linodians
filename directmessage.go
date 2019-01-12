package main

import (
	"fmt"
)

type targetParams struct {
	RecipientID string `json:"recipient_id"`
}

type messageDataParams struct {
	Text string `json:"text"`
}

type messageCreateParams struct {
	Target      targetParams      `json:"target"`
	MessageData messageDataParams `json:"message_data"`
}

type directMessageParams struct {
	Type          string              `json:"type"`
	MessageCreate messageCreateParams `json:"message_create"`
}

type directMessageEventParams struct {
	Event directMessageParams `json:"event"`
}

func sendDirectMessage(message string) error {
	params := directMessageEventParams{
		Event: directMessageParams{
			Type: "message_create",
			MessageCreate: messageCreateParams{
				Target: targetParams{
					RecipientID: c.DMTarget,
				},
				MessageData: messageDataParams{
					Text: message,
				},
			},
		},
	}
	resp, err := dmClient.New().Post("events/new.json").BodyJSON(params).ReceiveSuccess(nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("bad HTTP code: %d", resp.StatusCode)
	}
	return nil
}
