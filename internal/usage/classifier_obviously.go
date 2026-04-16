package usage

import (
	"context"
	"net/url"
	"slices"
	"strings"
)

// hostnameCategory groups hostnames with their classification metadata
type hostnameCategory struct {
	hostnames []string
	category  string   // e.g., "social_media", "news", "developer"
	tags      []string // default tags for this category
	reasoning string   // default reasoning template
}

// =============================================================================
// GROUP A: Always Block (regardless of path)
// These sites are distracting no matter what page you're on
// =============================================================================

var alwaysBlockCategories = []hostnameCategory{
	{
		category:  "social_media",
		reasoning: "Social media platform - high distraction potential",
		tags:      []string{"social_media", "distraction", "time_sink"},
		hostnames: []string{
			"facebook.com",
			"instagram.com",
			"snapchat.com",
			"threads.net",
			"tumblr.com",
		},
	},
	{
		category:  "communication",
		reasoning: "Personal communication channel - interrupts focus",
		tags:      []string{"communication", "personal", "distraction"},
		hostnames: []string{
			"telegram.org",
			"web.telegram.org",
			"whatsapp.com",
			"web.whatsapp.com",
			"messenger.com",
			"discord.com",
		},
	},
	{
		category:  "video_streaming",
		reasoning: "Video streaming platform - content consumption mode",
		tags:      []string{"video", "streaming", "entertainment", "time_sink"},
		hostnames: []string{
			"netflix.com",
			"disneyplus.com",
			"hulu.com",
			"hbomax.com",
			"max.com",
			"primevideo.com",
			"twitch.tv",
			"tiktok.com",
			"vimeo.com",
		},
	},
	{
		category:  "gaming",
		reasoning: "Gaming platform - entertainment and distraction",
		tags:      []string{"gaming", "entertainment", "time_sink"},
		hostnames: []string{
			"steampowered.com",
			"store.steampowered.com",
			"epicgames.com",
			"roblox.com",
			"ign.com",
			"kotaku.com",
			"gamespot.com",
			"chess.com",
			"lichess.org",
		},
	},
	{
		category:  "news",
		reasoning: "News site - information consumption that breaks focus",
		tags:      []string{"news", "media", "time_sink"},
		hostnames: []string{
			// US Major News Networks
			"abcnews.com",
			"abcnews.go.com",
			"ap.org",
			"associatedpress.com",
			"axios.com",
			"bloomberg.com",
			"businessinsider.com",
			"cbsnews.com",
			"cnbc.com",
			"cnn.com",
			"forbes.com",
			"foxnews.com",
			"marketwatch.com",
			"msnbc.com",
			"nbcnews.com",
			"news.yahoo.com",
			"newsweek.com",
			"nytimes.com",
			"politico.com",
			"propublica.org",
			"reuters.com",
			"thehill.com",
			"usatoday.com",
			"washingtonpost.com",
			"wsj.com",
			// US Regional & Local
			"chicagotribune.com",
			"dallasnews.com",
			"denverpost.com",
			"latimes.com",
			"miamiherald.com",
			"nypost.com",
			"philly.com",
			"sfchronicle.com",
			"tampabay.com",
			// US Magazines & Opinion
			"theatlantic.com",
			"thenewyorker.com",
			"time.com",
			"vox.com",
			// UK Major News
			"bbc.com",
			"bbc.co.uk",
			"dailymail.co.uk",
			"independent.co.uk",
			"mirror.co.uk",
			"sky.com",
			"telegraph.co.uk",
			"theguardian.com",
			"thesun.co.uk",
			"thetimes.co.uk",
			// UK Financial & Business
			"financialtimes.com",
			"ft.com",
			"theeconomist.com",
			// International English Language
			"aljazeera.com",
			"dw.com",
			"france24.com",
			"scmp.com",
			"theglobeandmail.com",
			"thehindu.com",
			// Tech News
			"arstechnica.com",
			"techcrunch.com",
			"theverge.com",
			"wired.com",
			// Sports News
			"espn.com",
			"theathletic.com",
		},
	},
	{
		category:  "shopping",
		reasoning: "Shopping site - consumption and distraction",
		tags:      []string{"shopping", "consumption", "time_sink"},
		hostnames: []string{
			"amazon.com",
			"amazon.co.uk",
			"amazon.de",
			"amazon.fr",
			"amazon.it",
			"amazon.es",
			"amazon.co.jp",
			"amazon.in",
			"amazon.com.au",
			"amazon.ca",
			"ebay.com",
			"etsy.com",
			"aliexpress.com",
			"walmart.com",
			"target.com",
			"bestbuy.com",
		},
	},
	{
		category:  "dating",
		reasoning: "Dating platform - personal distraction",
		tags:      []string{"dating", "personal", "time_sink"},
		hostnames: []string{
			"tinder.com",
			"okcupid.com",
			"match.com",
			"eharmony.com",
			"pof.com",
			"bumble.com",
			"hinge.co",
			"muzz.com",
		},
	},
	{
		category:  "adult",
		reasoning: "Adult content - blocked",
		tags:      []string{"adult", "blocked", "nsfw"},
		hostnames: []string{
			// Major Free Adult Video Sites
			"4tube.com",
			"beeg.com",
			"drtuber.com",
			"extremetube.com",
			"keezmovies.com",
			"porn.com",
			"porn300.com",
			"pornhd.com",
			"pornhub.com",
			"pornhubpremium.com",
			"pornmd.com",
			"pornoxo.com",
			"redtube.com",
			"spankwire.com",
			"sunporno.com",
			"tnaflix.com",
			"tube8.com",
			"xhamster.com",
			"xhamsterlive.com",
			"xnxx.com",
			"xvideos.com",
			"xvideos.red",
			"youporn.com",
			// Premium/Subscription Platforms
			"bongacams.com",
			"cam4.com",
			"camsoda.com",
			"chaturbate.com",
			"fansly.com",
			"imlive.com",
			"justfor.fans",
			"livejasmin.com",
			"manyvids.com",
			"myfreecams.com",
			"onlyfans.com",
			"streamate.com",
			"stripchat.com",
			// Premium Content Sites
			"bangbros.com",
			"brazzers.com",
			"digitalplayground.com",
			"fetishnetwork.com",
			"hustler.com",
			"kink.com",
			"mofos.com",
			"naughtyamerica.com",
			"penthouse.com",
			"playboy.com",
			"realitykings.com",
			"twistys.com",
			"vivid.com",
			"wicked.com",
			// Adult Dating & Hookup Sites
			"adultfriendfinder.com",
			"ashleymadison.com",
			"fabswingers.com",
			"fetlife.com",
			"grindr.com",
			"hornet.com",
			"scruff.com",
			// Adult Content Aggregators & Forums
			"4chan.org",
			"8chan.net",
			"imagefap.com",
			// Adult Comics & Hentai
			"e-hentai.org",
			"hanime.tv",
			"hentaihaven.org",
			"myreadingmanga.info",
			"nhentai.net",
		},
	},
}

// =============================================================================
// GROUP B: Block Exact Hostname Only
// Block homepage, but pass to LLM if there's a path (may contain productive content)
// =============================================================================

var exactBlockOnlyCategory = hostnameCategory{
	category:  "mixed_content_platform",
	reasoning: "Platform homepage is distracting, but specific pages may be productive",
	tags:      []string{"mixed_content", "conditional"},
	hostnames: []string{
		"x.com",
		"linkedin.com",
		"reddit.com",
	},
}

// =============================================================================
// GROUP C: Ambiguous on Exact Hostname
// Homepage is ambiguous - could lead to productive or distracting content
// Always pass to LLM if there's a path
// =============================================================================

var ambiguousHostnameCategory = hostnameCategory{
	category:  "ambiguous_platform",
	reasoning: "Ambiguous platform - homepage could lead to productive or distracting content",
	tags:      []string{"ambiguous", "mixed_content"},
	hostnames: []string{
		"youtube.com",
		"youtu.be",
		"pinterest.com",
		"medium.com",
		"news.ycombinator.com",
		"substack.com",
	},
}

// =============================================================================
// GROUP D: Always Productive (regardless of path)
// Developer tools, productivity apps, etc.
// =============================================================================

var alwaysProductiveCategories = []hostnameCategory{
	{
		category:  "developer_tools",
		reasoning: "Developer tool or documentation - productive work",
		tags:      []string{"development", "programming", "productive"},
		hostnames: []string{
			"github.com",
			"cursor.com",
			"gitlab.com",
			"bitbucket.org",
			"stackoverflow.com",
			"stackexchange.com",
			"dev.to",
			"w3schools.com",
			"developer.mozilla.org",
			"mdn.io",
			// Language documentation
			"golang.org",
			"go.dev",
			"pkg.go.dev",
			"react.dev",
			"reactjs.org",
			"vuejs.org",
			"angular.io",
			"svelte.dev",
			"typescriptlang.org",
			"python.org",
			"docs.python.org",
			"rust-lang.org",
			"doc.rust-lang.org",
			"ruby-lang.org",
			"php.net",
			"kotlinlang.org",
			"swift.org",
			// Package managers
			"npmjs.com",
			"yarnpkg.com",
			"pnpm.io",
			"pypi.org",
			"crates.io",
			"rubygems.org",
			"packagist.org",
			// Cloud & DevOps
			"docker.com",
			"docs.docker.com",
			"kubernetes.io",
			"aws.amazon.com",
			"docs.aws.amazon.com",
			"cloud.google.com",
			"azure.microsoft.com",
			"vercel.com",
			"netlify.com",
			"heroku.com",
			"digitalocean.com",
			"railway.app",
			"render.com",
			"fly.io",
			// Project management
			"linear.app",
			"jira.atlassian.com",
			"atlassian.com",
			"trello.com",
			"asana.com",
			"clickup.com",
			"monday.com",
			"notion.so",
			// Design tools
			"figma.com",
			"sketch.com",
			"zeplin.io",
			// Local development
			"localhost",
			"127.0.0.1",
		},
	},
	{
		category:  "productivity_tools",
		reasoning: "Productivity tool - work-related activity",
		tags:      []string{"productivity", "work", "tools"},
		hostnames: []string{
			// Anti-embarrassment, imgine self-blocking
			"focusd.so",
			// Google Workspace
			"drive.google.com",
			"docs.google.com",
			"sheets.google.com",
			"slides.google.com",
			"calendar.google.com",
			"mail.google.com",
			"meet.google.com",
			// Microsoft 365
			"outlook.live.com",
			"outlook.office.com",
			"outlook.office365.com",
			"teams.microsoft.com",
			"onedrive.live.com",
			"office.com",
			// Video conferencing
			"zoom.us",
			"webex.com",
			// Cloud storage
			"dropbox.com",
			"box.com",
			// Collaboration
			"miro.com",
			"airtable.com",
			"coda.io",
			// AI assistants
			"chatgpt.com",
			"chat.openai.com",
			"claude.ai",
			"openai.com",
			"anthropic.com",
			"bard.google.com",
			"gemini.google.com",
			// Design
			"canva.com",
		},
	},
}

// =============================================================================
// GROUP E: Block Specific Paths on Mixed-Content Platforms
// These URL prefixes are always distracting, even though the domain
// itself may have productive content on other paths
// =============================================================================

var alwaysBlockPathCategories = []hostnameCategory{
	{
		category:  "social_media_non_content",
		reasoning: "Non-content page on mixed platform - distracting",
		tags:      []string{"social_media", "distraction", "time_sink"},
		hostnames: []string{
			"linkedin.com/feed",
			"linkedin.com/messaging",
			"linkedin.com/mynetwork",
			"linkedin.com/jobs",
			"linkedin.com/notifications",
			"linkedin.com/search",
			"x.com/home",
			"x.com/explore",
			"x.com/notifications",
			"x.com/messages",
			"x.com/search",
			"x.com/i/trending",
		},
	},
}

// =============================================================================
// GROUP F: Always Allow Specific Paths
// These URL prefixes are always allowed (productive), e.g. app's own pages
// =============================================================================

var alwaysAllowPathCategories = []hostnameCategory{
	{
		category:  "focus_app",
		reasoning: "Focus app page - part of focus experience",
		tags:      []string{"focus_app", "productive", "tools"},
		hostnames: []string{
			"focusd.so/blocked",
		},
	},
}

// ObviousClassification is a classifier that uses obvious rules to classify websites or applications
//
// There are 4 main classification principles for websites:
//  1. Obviously distracting websites (e.g. social media, news, shopping, etc.)
//  2. Obviously productive websites (e.g. developer tools, productivity apps, etc.)
//  3. Ambiguous hostnames (YouTube, Reddit, etc.) - depending on the URL and title it can be either distracting or productive
//     so they should be passed to the LLM for analysis for better accuracy
//  4. Website whos homepage is distracting, but specific pages may be productive like x.com or linkedin.com which have
//     content related to doing work or research so they should be passed to the LLM for analysis for better accuracy
//
// There are 3 main classification principles for applications:
//  1. Obviously distracting applications (e.g. social media, news, shopping, etc.)
//  2. Obviously productive applications (e.g. developer tools, productivity apps, etc.)
//  3. Applications that are used mixed of productive and distracting content depending on the usage context
//     eg. Slack - funny dogs videos is distracting, but billing-internal-channel is supporting for communication
func (s *Service) classifyObviously(ctx context.Context, appName string, url *url.URL) (*ObviouslyClassificationResult, error) {
	if url != nil {
		return s.classifyObviouslyWebsite(ctx, url)
	}

	return s.classifyObviouslyApplication(ctx, appName)
}

func (s *Service) classifyObviouslyWebsite(ctx context.Context, browserURL *url.URL) (*ObviouslyClassificationResult, error) {
	hostname := browserURL.Hostname()
	path := browserURL.Path

	hasPath := path != "" && path != "/"

	if isDeterministicCriticalNoBlockURL(browserURL) {
		return &ObviouslyClassificationResult{
			BasicClassificationResult: BasicClassificationResult{
				Classification: ClassificationNeutral,
				ClassificationReason: "Payment/booking flow detected - safety override to avoid interruption",
				Tags:           []string{"other"},
			},
		}, nil
	}

	// 0. Check always-allow path categories first (e.g. focusd.so/blocked)
	fullURL := hostname + path
	for _, cat := range alwaysAllowPathCategories {
		if slices.ContainsFunc(cat.hostnames, func(prefix string) bool {
			return strings.HasPrefix(fullURL, prefix)
		}) {
			return &ObviouslyClassificationResult{
				BasicClassificationResult: BasicClassificationResult{
					Classification: ClassificationProductive,
					ClassificationReason: cat.reasoning,
					Tags:           cat.tags,
				},
			}, nil
		}
	}

	// Helper: check if hostname matches any in the list (exact or subdomain)
	matchesAny := func(list []string) bool {
		return slices.Contains(list, hostname) || slices.ContainsFunc(list, func(h string) bool {
			return strings.HasSuffix(hostname, "."+h)
		})
	}

	// Helper: check if hostname is an exact match (no subdomain matching)
	exactMatch := func(list []string) bool {
		return slices.Contains(list, hostname)
	}

	// 1. Check ambiguous hostnames first (YouTube, Reddit, etc.)
	// If exact hostname match with no path, return Neutral
	// If has path, pass to LLM for analysis
	if exactMatch(ambiguousHostnameCategory.hostnames) {
		if hasPath {
			// Has a specific path - let LLM analyze the content
			return nil, nil
		}
		// Exact homepage - ambiguous, could go either way
		return &ObviouslyClassificationResult{
			BasicClassificationResult: BasicClassificationResult{
				Classification: ClassificationNeutral,
				ClassificationReason: ambiguousHostnameCategory.reasoning,
				Tags:           ambiguousHostnameCategory.tags,
			},
		}, nil
	}

	// 2. Check exact-block-only hostnames (X/Twitter, LinkedIn)
	// Block only if exact hostname with no path
	// If has path, pass to LLM (might be a productive article)
	if exactMatch(exactBlockOnlyCategory.hostnames) {
		if hasPath {
			fullURL := hostname + path
			for _, cat := range alwaysBlockPathCategories {
				if slices.ContainsFunc(cat.hostnames, func(prefix string) bool {
					return strings.HasPrefix(fullURL, prefix)
				}) {
					return &ObviouslyClassificationResult{
						BasicClassificationResult: BasicClassificationResult{
							Classification: ClassificationDistracting,
							ClassificationReason: cat.reasoning,
							Tags:           cat.tags,
						},
					}, nil
				}
			}
			// Has a specific path - let LLM analyze the content
			return nil, nil
		}
		// Exact homepage - distracting
		return &ObviouslyClassificationResult{
			BasicClassificationResult: BasicClassificationResult{
				Classification: ClassificationDistracting,
				ClassificationReason: exactBlockOnlyCategory.reasoning,
				Tags:           exactBlockOnlyCategory.tags,
			},
		}, nil
	}

	// 3. Check always-block categories (social media, news, etc.)
	// Block regardless of path
	for _, cat := range alwaysBlockCategories {
		if matchesAny(cat.hostnames) {
			return &ObviouslyClassificationResult{
				BasicClassificationResult: BasicClassificationResult{
					Classification: ClassificationDistracting,
					ClassificationReason: cat.reasoning,
					Tags:           cat.tags,
				},
			}, nil
		}
	}

	// 4. Check always-productive categories (developer tools, productivity)
	// Allow regardless of path
	for _, cat := range alwaysProductiveCategories {
		if matchesAny(cat.hostnames) {
			return &ObviouslyClassificationResult{
				BasicClassificationResult: BasicClassificationResult{
					Classification: ClassificationProductive,
					ClassificationReason: cat.reasoning,
					Tags:           cat.tags,
				},
			}, nil
		}
	}

	// 5. Unknown hostname - return nil to pass to LLM for analysis
	return nil, nil
}

func (s *Service) classifyObviouslyApplication(ctx context.Context, appName string) (*ObviouslyClassificationResult, error) {
	return nil, nil
}
