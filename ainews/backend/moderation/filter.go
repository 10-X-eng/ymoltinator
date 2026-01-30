package moderation

import (
	"regexp"
	"strings"
	"sync"
)

// Filter provides content moderation functionality
type Filter struct {
	badWords    map[string]bool
	patterns    []*regexp.Regexp
	mu          sync.RWMutex
}

// NewFilter creates a new content filter with comprehensive bad word list
func NewFilter() *Filter {
	f := &Filter{
		badWords: make(map[string]bool),
	}
	f.loadBadWords()
	f.compilePatterns()
	return f
}

// loadBadWords loads the comprehensive list of prohibited words
func (f *Filter) loadBadWords() {
	// Comprehensive list of profanity, slurs, hate speech, and inappropriate content
	words := []string{
		// Profanity - Common
		"fuck", "fucking", "fucked", "fucker", "fuckers", "fucks", "fuckhead", "fuckface",
		"motherfucker", "motherfucking", "motherfuckers",
		"shit", "shits", "shitty", "shitting", "bullshit", "horseshit", "dipshit", "shithead",
		"ass", "asshole", "assholes", "asses", "asshat", "assface", "dumbass", "fatass", "jackass",
		"bitch", "bitches", "bitchy", "bitching", "sonofabitch",
		"damn", "damned", "damnit", "goddamn", "goddamnit",
		"crap", "crappy",
		"piss", "pissed", "pissing",
		"hell", "hellish",
		"bastard", "bastards",
		"cunt", "cunts",
		"cock", "cocks", "cocksucker", "cocksuckers",
		"dick", "dicks", "dickhead", "dickheads", "dickface",
		"pussy", "pussies",
		"twat", "twats",
		"whore", "whores", "whorish",
		"slut", "sluts", "slutty",
		"prick", "pricks",
		
		// Racial slurs and hate speech (comprehensive but abbreviated for common variations)
		"nigger", "niggers", "nigga", "niggas", "negro", "negroes",
		"chink", "chinks",
		"gook", "gooks",
		"spic", "spics", "spick", "spicks",
		"wetback", "wetbacks",
		"beaner", "beaners",
		"kike", "kikes",
		"hymie", "hymies",
		"cracker", "crackers",
		"honky", "honkies", "honkey",
		"gringo", "gringos",
		"wop", "wops",
		"dago", "dagos",
		"polack", "polacks",
		"kraut", "krauts",
		"jap", "japs",
		"paki", "pakis",
		"towelhead", "towelheads",
		"raghead", "ragheads",
		"camel jockey",
		"sandnigger", "sandniggers",
		"coon", "coons",
		"darkie", "darkies",
		"jungle bunny",
		"porch monkey",
		"moon cricket",
		"tar baby",
		"uncle tom",
		"house negro",
		"oreo",
		"banana",
		"coconut",
		"redskin", "redskins",
		"injun", "injuns",
		"squaw", "squaws",
		"halfbreed",
		"mudblood",
		"zipperhead",
		"slope", "slopes",
		"chinaman",
		
		// LGBTQ+ slurs
		"fag", "fags", "faggot", "faggots", "faggy",
		"dyke", "dykes",
		"homo", "homos",
		"queer", "queers", // Note: this has been reclaimed by some
		"tranny", "trannies",
		"shemale", "shemales",
		"he-she",
		"ladyboy", "ladyboys",
		"sissy", "sissies",
		"pansy", "pansies",
		"fairy", "fairies", // in slur context
		"fruit", "fruits", // in slur context
		"pillow biter",
		"butt pirate",
		"carpet muncher",
		"muff diver",
		"rump ranger",
		
		// Ableist slurs
		"retard", "retards", "retarded", "tard", "tards",
		"spaz", "spazz", "spastic",
		"cripple", "cripples", "crippled",
		"lame",
		"moron", "morons", "moronic",
		"idiot", "idiots", "idiotic",
		"imbecile", "imbeciles",
		"cretin", "cretins",
		"mongoloid", "mongoloids",
		"psycho", "psychos",
		"lunatic", "lunatics",
		"nutjob", "nutjobs",
		"schizo",
		"mental case",
		"window licker",
		
		// Body/appearance insults (severe)
		"fatso", "fatty", "fatties",
		"lardass",
		"porker", "porkers",
		"blimp",
		"whale", // in insult context
		"pig", "pigs", // in insult context
		"midget", "midgets",
		"dwarf", "dwarfs", // when used as insult
		"freak", "freaks",
		"ugly",
		"fugly",
		
		// Sexual content
		"porn", "porno", "pornography",
		"xxx",
		"nude", "nudes", "nudity",
		"naked",
		"sex", "sexual", "sexually",
		"erotic", "erotica",
		"orgasm", "orgasms", "orgasmic",
		"masturbate", "masturbating", "masturbation",
		"ejaculate", "ejaculating", "ejaculation",
		"cum", "cumming", "cumshot",
		"semen", "sperm",
		"dildo", "dildos",
		"vibrator", "vibrators",
		"blowjob", "blowjobs", "bj", "bjs",
		"handjob", "handjobs", "hj",
		"rimjob", "rimjobs",
		"titjob",
		"footjob",
		"anal", // in sexual context
		"oral", // in sexual context
		"fellatio",
		"cunnilingus",
		"sodomy",
		"bestiality",
		"incest",
		"pedophile", "pedophiles", "pedophilia", "pedo", "pedos",
		"rape", "raping", "rapist", "rapists", "raped",
		"molest", "molesting", "molester", "molestation",
		"penis", "penises",
		"vagina", "vaginas",
		"tits", "titties", "titty",
		"boobs", "boobies", "booby",
		"nipple", "nipples",
		"clitoris", "clit",
		"labia",
		"scrotum",
		"testicle", "testicles", "balls",
		"boner", "boners",
		"erection", "erections",
		"horny", "horniness",
		"aroused", "arousal",
		"kinky", "kink",
		"fetish", "fetishes",
		"bdsm",
		"bondage",
		"dominatrix",
		"stripper", "strippers",
		"prostitute", "prostitutes", "prostitution",
		"hooker", "hookers",
		"escort", // in sexual services context
		"brothel", "brothels",
		"pimp", "pimps", "pimping",
		"gigolo",
		"camgirl", "camgirls",
		"onlyfans",
		"bangbros",
		"pornhub",
		"xvideos",
		"xhamster",
		"redtube",
		"youporn",
		"brazzers",
		
		// Violence and threats
		"kill", "killing", "killer", "killers", "killed",
		"murder", "murdering", "murderer", "murderers", "murdered",
		"die", "dying", "death",
		"suicide", "suicidal",
		"bomb", "bombs", "bombing", "bomber",
		"terrorist", "terrorists", "terrorism", "terror",
		"attack", "attacks", "attacking",
		"shoot", "shooting", "shooter", "shooters",
		"stab", "stabbing", "stabbed",
		"strangle", "strangling", "strangled",
		"torture", "torturing", "tortured",
		"massacre", "massacred",
		"slaughter", "slaughtered",
		"genocide",
		"holocaust",
		"execute", "executing", "executed", "execution",
		"assassinate", "assassinating", "assassination", "assassin",
		"decapitate", "decapitation", "beheading",
		"dismember", "dismembered",
		"mutilate", "mutilated", "mutilation",
		"bloodbath",
		"carnage",
		
		// Drugs
		"cocaine", "coke",
		"heroin", "heroine", // drug context
		"meth", "methamphetamine", "crystal meth",
		"crack", // drug context
		"ecstasy", "mdma", "molly",
		"lsd", "acid", // drug context
		"marijuana", "weed", "pot", "cannabis", "ganja",
		"dope",
		"hash", "hashish",
		"opium",
		"fentanyl",
		"oxy", "oxycontin",
		"xanax",
		"valium",
		"ketamine",
		"pcp",
		"shrooms", "mushrooms", // drug context
		"drug dealer", "drug dealing",
		"overdose", "od",
		"junkie", "junkies",
		"crackhead", "crackheads",
		"pothead", "potheads",
		"stoner", "stoners",
		"druggie", "druggies",
		
		// Hate groups and symbols
		"nazi", "nazis", "nazism", "neo-nazi",
		"kkk", "ku klux klan",
		"white power",
		"white supremacy", "white supremacist",
		"aryan",
		"skinhead", "skinheads",
		"heil hitler",
		"seig heil", "sieg heil",
		"swastika",
		"14 words", "fourteen words",
		"1488",
		"blood and soil",
		"jews will not replace us",
		"great replacement",
		"white genocide",
		"race war",
		"ethnic cleansing",
		
		// Conspiracy/harmful misinformation keywords
		"qanon",
		"pizzagate",
		"deep state",
		"false flag",
		"crisis actor",
		"adrenochrome",
		"chemtrails",
		"flat earth",
		
		// Scam/spam indicators
		"nigerian prince",
		"wire transfer",
		"advance fee",
		"get rich quick",
		"make money fast",
		"work from home", // often spam
		"crypto giveaway",
		"double your bitcoin",
		"investment opportunity",
		"limited time offer",
		"act now",
		"call now",
		"click here",
		"free money",
		"congratulations you won",
		"you have been selected",
		"claim your prize",
		
		// Miscellaneous offensive
		"kys", // kill yourself
		"neck yourself",
		"go die",
		"drink bleach",
		"eat shit",
		"suck my",
		"blow me",
		"screw you",
		"f you", "fu", "f u",
		"stfu",
		"gtfo",
		"lmfao",
		"wtf",
		"milf",
		"gilf",
		"dilf",
		"thot", "thots",
		"simp", "simps", "simping",
		"incel", "incels",
		"cuck", "cucks", "cuckold",
		"beta male",
		"soy boy", "soyboy",
		"neckbeard", "neckbeards",
		"karen", "karens", // when used as slur
		"boomer", "boomers", // when used pejoratively
		"ok boomer",
		"triggered",
		"snowflake", "snowflakes",
		"libtard", "libtards",
		"conservatard",
		"demonrat", "demonrats",
		"repugnican", "repugnicans",
		"feminazi", "feminazis",
		"sjw", "sjws",
		"woke",
		"cancel culture",
	}
	
	for _, word := range words {
		f.badWords[strings.ToLower(word)] = true
	}
}

// compilePatterns compiles regex patterns for more sophisticated detection
func (f *Filter) compilePatterns() {
	patterns := []string{
		// Leetspeak variations
		`f[u|v|\*|@|0]ck`,
		`sh[i|1|\*|@]t`,
		`[a|@|\*]ss`,
		`b[i|1|\*]tch`,
		`n[i|1|\*]gg[a|e|@]r?s?`,
		`f[a|@]g+[o|0]?t?s?`,
		`c[u|v|\*]nt`,
		`d[i|1|\*]ck`,
		`p[u|v|\*]ssy`,
		`wh[o|0]re`,
		`sl[u|v]t`,
		
		// Common obfuscations with special chars
		`f[\.\-\_\*]?u[\.\-\_\*]?c[\.\-\_\*]?k`,
		`s[\.\-\_\*]?h[\.\-\_\*]?i[\.\-\_\*]?t`,
		`a[\.\-\_\*]?s[\.\-\_\*]?s`,
		
		// Spaced out profanity
		`f\s+u\s+c\s+k`,
		`s\s+h\s+i\s+t`,
		`b\s+i\s+t\s+c\s+h`,
		
		// Common misspellings/evasions
		`phuck`,
		`phuk`,
		`fuk`,
		`fvck`,
		`azz`,
		`a\$\$`,
		`b1tch`,
		`sh1t`,
		`d1ck`,
		`c0ck`,
		`p0rn`,
		
		// URL patterns for adult sites
		`(?i)(porn|xxx|adult|sex|nude|naked|erotic)\.(com|net|org|xxx|adult)`,
	}
	
	f.patterns = make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		if re, err := regexp.Compile(`(?i)` + pattern); err == nil {
			f.patterns = append(f.patterns, re)
		}
	}
}

// ContainsBadWords checks if the text contains any prohibited content
// Returns true if bad content found, along with the matched word/pattern
func (f *Filter) ContainsBadWords(text string) (bool, string) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	lowerText := strings.ToLower(text)
	
	// Check word boundaries for exact matches
	words := tokenize(lowerText)
	for _, word := range words {
		if f.badWords[word] {
			return true, word
		}
	}
	
	// Check regex patterns
	for _, pattern := range f.patterns {
		if match := pattern.FindString(lowerText); match != "" {
			return true, match
		}
	}
	
	return false, ""
}

// tokenize splits text into words, handling various separators
func tokenize(text string) []string {
	// Replace common separators with spaces
	replacer := strings.NewReplacer(
		".", " ", ",", " ", "!", " ", "?", " ",
		";", " ", ":", " ", "'", " ", "\"", " ",
		"(", " ", ")", " ", "[", " ", "]", " ",
		"{", " ", "}", " ", "/", " ", "\\", " ",
		"|", " ", "\n", " ", "\r", " ", "\t", " ",
		"-", " ", "_", " ",
	)
	cleaned := replacer.Replace(text)
	
	// Split and filter empty strings
	parts := strings.Fields(cleaned)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) > 0 {
			result = append(result, part)
		}
	}
	return result
}

// ValidateContent validates both title and content of a story
func (f *Filter) ValidateContent(title, content, url string) (bool, string) {
	// Check title
	if found, word := f.ContainsBadWords(title); found {
		return false, "Title contains prohibited content: " + word
	}
	
	// Check content
	if found, word := f.ContainsBadWords(content); found {
		return false, "Content contains prohibited content: " + word
	}
	
	// Check URL
	if found, word := f.ContainsBadWords(url); found {
		return false, "URL contains prohibited content: " + word
	}
	
	return true, ""
}
