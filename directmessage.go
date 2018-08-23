package main

type targetParams struct {
	RecipientID string `url:"recipient_id"`
}

type messageCreateParams struct {
	Target      targetParams      `url:"target"`
	MessageData map[string]string `url:"message_data"`
}

type directMessageParams struct {
	Type          string              `url:"type"`
	MessageCreate messageCreateParams `url:"message_create"`
}

func sendDirectMessage(message string) error {
	params := directMessageParams{
		Type: "message_create",
		MessageCreate: messageCreateParams{
			Target: targetParams{
				RecipientID: c.DMTarget,
			},
			MessageData: map[string]string{
				"text": message,
			},
		},
	}
	_, err := dmClient.New().Get("events/new.json").QueryStruct(params).ReceiveSuccess(nil)
	return err
}
