package pkg

import (
	"encoding/json"
	"fmt"
)

type WhatsAppMessage struct {
	ChatID  string `json:"chatId"`
	Message string `json:"message"`
}

func SendWhatsAppMessage(apiURL, chatID, message string) error {
	fmt.Printf("Sending WhatsApp message to %s\n", chatID)
	msgBody := WhatsAppMessage{
		ChatID:  chatID,
		Message: message,
	}

	bodyJSON, err := json.Marshal(msgBody)
	if err != nil {
		return fmt.Errorf("error marshaling message: %v", err)
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	data, err := FetchJS(apiURL, "POST", headers, string(bodyJSON))
	if err != nil {
		return fmt.Errorf("error sending WhatsApp message: %v", err)
	}

	fmt.Printf("WhatsApp message sent successfully: %s\n", string(data))
	return nil
}
