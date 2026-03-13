package usage

var (
	instructionGenericWebsiteClassification    = mustLoadPrompt("website", "website_generic.txt")
	instructionYouTubeWebsiteClassification    = mustLoadPrompt("website", "website_youtube.txt")
	instructionRedditWebsiteClassification     = mustLoadPrompt("website", "website_reddit.txt")
	instructionLinkedInWebsiteClassification   = mustLoadPrompt("website", "website_linkedin.txt")
	instructionMediumWebsiteClassification     = mustLoadPrompt("website", "website_medium.txt")
	instructionTwitterWebsiteClassification    = mustLoadPrompt("website", "website_twitter.txt")
	instructionHackerNewsWebsiteClassification = mustLoadPrompt("website", "website_hackernews.txt")
	instructionSubstackWebsiteClassification   = mustLoadPrompt("website", "website_substack.txt")
)
