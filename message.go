package main

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ErrResponseSent for when a response has already been sent to an interaction
var ErrResponseSent = errors.New("response has already been sent")

// Response wraps a discord interaction to simplify sending messages
//
// This struct will automatically send a deferral is the response has taken
// more than 2 seconds to create.
//
// Flags are supported oddly, at the creation of the response 'default flags' can be set
// and they are applied when the deferral is sent. However, if a message is sent before the
// deferral then a new set of flags can be passed
type Response struct {
	sync.Mutex

	cancel context.CancelFunc

	session     *discordgo.Session
	interaction *discordgo.Interaction

	deferred bool
	finished bool

	defaultFlags discordgo.MessageFlags
}

// NewResponse creates a new response handler for an interaction
func NewResponse(s *discordgo.Session, i *discordgo.Interaction, flags discordgo.MessageFlags) *Response {
	ctx, cancel := context.WithCancel(context.Background())

	r := &Response{
		session:      s,
		interaction:  i,
		cancel:       cancel,
		defaultFlags: flags,
	}

	go func() {
		select {
		// Send a deferral if the response has taken more than 1.5 seconds
		case <-time.After(2 * time.Second):
			r.Defer()
		// End the goroutine if the response has been completed
		case <-ctx.Done():
		}
	}()

	return r
}

// Defer defers the response to the interaction to allow for longer processing times
func (r *Response) Defer() error {
	r.Lock()
	defer r.Unlock()

	if r.deferred {
		return nil
	}

	r.deferred = true
	return r.session.InteractionRespond(r.interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: r.defaultFlags,
		},
	})
}

// TextMessage responds to an interaction with a simple text message
func (r *Response) TextMessage(content string, flags discordgo.MessageFlags) error {
	r.Lock()
	defer r.Unlock()

	if r.finished {
		return ErrResponseSent
	}

	r.cancel()
	r.finished = true

	if r.deferred {
		_, err := r.session.InteractionResponseEdit(r.interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return err
	}

	return r.session.InteractionRespond(r.interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   flags,
		},
	})
}

// EmbedMessage responds to an interaction with an embed message, and files
func (r *Response) EmbedMessage(embed *discordgo.MessageEmbed, file *discordgo.File, flags discordgo.MessageFlags) error {
	r.Lock()
	defer r.Unlock()

	if r.finished {
		return ErrResponseSent
	}

	r.cancel()
	r.finished = true

	if r.deferred {
		_, err := r.session.InteractionResponseEdit(r.interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
			Files:  []*discordgo.File{file},
		})
		return err
	}

	return r.session.InteractionRespond(r.interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Files:  []*discordgo.File{file},
			Flags:  flags,
		},
	})
}
