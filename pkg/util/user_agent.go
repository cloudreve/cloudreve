package util

import "strings"

// IsSocialMediaBot checks if the User-Agent belongs to a social media crawler.
func IsSocialMediaBot(ua string) bool {
	bots := []string{
		"facebookexternalhit",
		"twitterbot",
		"linkedinbot",
		"whatsapp",
		"slackbot",
		"discordbot",
		"telegrambot",
		"facebot",
	}
	ua = strings.ToLower(ua)
	for _, bot := range bots {
		if strings.Contains(ua, bot) {
			return true
		}
	}
	return false
}
