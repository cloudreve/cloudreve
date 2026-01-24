package util

import "strings"

// IsSocialMediaBot checks if the User-Agent belongs to a social media crawler.
func IsSocialMediaBot(ua string) bool {
	bots := []string{
		// Meta (Facebook, Instagram, WhatsApp)
		"facebookexternalhit",
		"facebookcatalog",
		"facebot",
		"meta-externalagent",
		// Twitter/X
		"twitterbot",
		// LinkedIn
		"linkedinbot",
		// Discord
		"discordbot",
		// Telegram
		"telegrambot",
		// Slack
		"slackbot",
		// WhatsApp
		"whatsapp",
		// Pinterest
		"pinterestbot",
	}
	ua = strings.ToLower(ua)
	for _, bot := range bots {
		if strings.Contains(ua, bot) {
			return true
		}
	}
	return false
}
