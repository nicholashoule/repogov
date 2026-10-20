package repogov

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Language describes a natural (human) language that [Config.Language] may
// name: the language of a repository's prose -- Markdown and text files and
// code comments -- not a programming language.
type Language struct {
	// Tag is the canonical BCP 47 tag, e.g. "en-US" or "zh-Hans".
	Tag string

	// Name is the English display name, e.g. "American English".
	Name string

	// Convention names the spelling or script convention that distinguishes
	// this variant from its siblings (e.g. "American spelling: color,
	// behavior, -ize"). Empty when the language has no regional variant in
	// the known set.
	Convention string

	// RTL reports whether the language is written right to left.
	RTL bool
}

// Directive returns the instruction line written into scaffolded agent files
// so an AI agent reading them honors the prose language, e.g.
// "Prose language: American English (en-US). Write documentation (.md,
// .txt), code comments and commit messages in American English, ...".
func (l Language) Directive() string {
	conv := "with its spelling and conventions"
	if l.Convention != "" {
		conv += " (" + l.Convention + ")"
	}
	s := "Prose language: " + l.Name + " (" + l.Tag + "). Write documentation (.md, .txt), code comments and commit messages in " +
		l.Name + ", " + conv + "; keep identifiers, code and quoted text as they are."
	if l.RTL {
		s += " The script is right-to-left; keep code, paths and identifiers in left-to-right runs."
	}
	return s
}

// knownLanguages is the closed set of prose languages, in documented order.
var knownLanguages = []Language{
	{Tag: "en-US", Name: "American English", Convention: "American spelling: color, behavior, -ize"},
	{Tag: "en-GB", Name: "British English", Convention: "British spelling: colour, behaviour, -ise"},       //nolint:misspell // intentional non-US spelling illustrates the variant
	{Tag: "en-AU", Name: "Australian English", Convention: "Australian spelling: colour, behaviour, -ise"}, //nolint:misspell // intentional non-US spelling illustrates the variant
	{Tag: "en-CA", Name: "Canadian English", Convention: "Canadian spelling: colour, behaviour, -ize"},     //nolint:misspell // intentional non-US spelling illustrates the variant
	{Tag: "en-IN", Name: "Indian English", Convention: "Indian English spelling: colour, behaviour, -ise"}, //nolint:misspell // intentional non-US spelling illustrates the variant
	{Tag: "es-ES", Name: "European Spanish", Convention: "Spain usage, vosotros"},
	{Tag: "es-MX", Name: "Mexican Spanish", Convention: "Mexican usage, ustedes"},
	{Tag: "es-419", Name: "Latin American Spanish", Convention: "Latin American usage, ustedes"},
	{Tag: "pt-BR", Name: "Brazilian Portuguese", Convention: "Brazilian spelling and vocabulary, not European Portuguese"},
	{Tag: "pt-PT", Name: "European Portuguese", Convention: "European spelling and vocabulary, not Brazilian Portuguese"},
	{Tag: "fr-FR", Name: "French", Convention: "France usage"},
	{Tag: "fr-CA", Name: "Canadian French", Convention: "Quebec usage and vocabulary"},
	{Tag: "de-DE", Name: "German", Convention: "Germany usage, with ß"},
	{Tag: "de-AT", Name: "Austrian German", Convention: "Austrian usage and vocabulary"},
	{Tag: "de-CH", Name: "Swiss Standard German", Convention: "Swiss spelling: ss instead of ß"},
	{Tag: "it-IT", Name: "Italian"},
	{Tag: "nl-NL", Name: "Dutch"},
	{Tag: "sv-SE", Name: "Swedish"},
	{Tag: "nb-NO", Name: "Norwegian Bokmål", Convention: "Bokmål written standard, not Nynorsk"},
	{Tag: "da-DK", Name: "Danish"},
	{Tag: "fi-FI", Name: "Finnish"},
	{Tag: "is-IS", Name: "Icelandic"},
	{Tag: "pl-PL", Name: "Polish"},
	{Tag: "cs-CZ", Name: "Czech"},
	{Tag: "sk-SK", Name: "Slovak"},
	{Tag: "hu-HU", Name: "Hungarian"},
	{Tag: "ro-RO", Name: "Romanian"},
	{Tag: "bg-BG", Name: "Bulgarian"},
	{Tag: "hr-HR", Name: "Croatian"},
	{Tag: "sr-RS", Name: "Serbian", Convention: "Cyrillic script"},
	{Tag: "sl-SI", Name: "Slovenian"},
	{Tag: "uk-UA", Name: "Ukrainian"},
	{Tag: "ru-RU", Name: "Russian"},
	{Tag: "el-GR", Name: "Greek"},
	{Tag: "tr-TR", Name: "Turkish"},
	{Tag: "lt-LT", Name: "Lithuanian"},
	{Tag: "lv-LV", Name: "Latvian"},
	{Tag: "et-EE", Name: "Estonian"},
	{Tag: "he-IL", Name: "Hebrew", RTL: true},
	{Tag: "ar-SA", Name: "Arabic", Convention: "Modern Standard Arabic", RTL: true},
	{Tag: "fa-IR", Name: "Persian", RTL: true},
	{Tag: "ur-PK", Name: "Urdu", RTL: true},
	{Tag: "hi-IN", Name: "Hindi"},
	{Tag: "bn-BD", Name: "Bengali"},
	{Tag: "ta-IN", Name: "Tamil"},
	{Tag: "te-IN", Name: "Telugu"},
	{Tag: "mr-IN", Name: "Marathi"},
	{Tag: "ja-JP", Name: "Japanese"},
	{Tag: "ko-KR", Name: "Korean"},
	{Tag: "zh-Hans", Name: "Simplified Chinese", Convention: "Simplified characters, not Traditional"},
	{Tag: "zh-Hant", Name: "Traditional Chinese", Convention: "Traditional characters, not Simplified"},
	{Tag: "th-TH", Name: "Thai"},
	{Tag: "vi-VN", Name: "Vietnamese"},
	{Tag: "id-ID", Name: "Indonesian"},
	{Tag: "ms-MY", Name: "Malay"},
	{Tag: "fil-PH", Name: "Filipino"},
	{Tag: "sw-KE", Name: "Swahili"},
	{Tag: "af-ZA", Name: "Afrikaans"},
	{Tag: "ca-ES", Name: "Catalan"},
	{Tag: "eu-ES", Name: "Basque"},
	{Tag: "gl-ES", Name: "Galician"},
	{Tag: "cy-GB", Name: "Welsh"},
	{Tag: "ga-IE", Name: "Irish"},
}

// languageAliases maps lowercase alternate tags to canonical tags. Bare
// language subtags resolve to the documented default variant (en -> en-US,
// es -> es-ES, pt -> pt-BR, fr -> fr-FR, de -> de-DE, zh -> zh-Hans, ...);
// every other bare subtag resolves to its only variant.
var languageAliases = map[string]string{
	"en":    "en-US",
	"es":    "es-ES",
	"pt":    "pt-BR",
	"fr":    "fr-FR",
	"de":    "de-DE",
	"zh":    "zh-Hans",
	"zh-cn": "zh-Hans",
	"zh-sg": "zh-Hans",
	"zh-tw": "zh-Hant",
	"zh-hk": "zh-Hant",
	"zh-mo": "zh-Hant",
	"no":    "nb-NO",
	"nb":    "nb-NO",
	"iw":    "he-IL",
	"tl":    "fil-PH",
}

// languageIndex maps lowercase canonical tags and aliases to an index into
// knownLanguages.
var languageIndex = func() map[string]int {
	idx := make(map[string]int, len(knownLanguages)*2)
	primaryCount := make(map[string]int)
	for i, l := range knownLanguages {
		idx[strings.ToLower(l.Tag)] = i
		primaryCount[primarySubtag(l.Tag)]++
	}
	// A bare subtag with a single known variant resolves to it (ja -> ja-JP).
	for i, l := range knownLanguages {
		if p := primarySubtag(l.Tag); primaryCount[p] == 1 {
			if _, taken := idx[p]; !taken {
				idx[p] = i
			}
		}
	}
	for alias, tag := range languageAliases {
		idx[alias] = idx[strings.ToLower(tag)]
	}
	return idx
}()

// primarySubtag returns the lowercase primary language subtag of tag
// ("en" for "en-US").
func primarySubtag(tag string) string {
	tag = strings.ToLower(strings.ReplaceAll(tag, "_", "-"))
	if i := strings.IndexByte(tag, '-'); i >= 0 {
		return tag[:i]
	}
	return tag
}

// KnownLanguages returns the closed set of prose languages accepted by
// [Config.Language], in documented order. The slice is a copy; callers may
// modify it.
func KnownLanguages() []Language {
	out := make([]Language, len(knownLanguages))
	copy(out, knownLanguages)
	return out
}

// LookupLanguage resolves a language tag to its [Language]. Matching is
// case-insensitive, accepts "_" in place of "-", and resolves aliases and
// bare subtags to their documented default (see [NormalizeLanguage]).
func LookupLanguage(tag string) (Language, bool) {
	key := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(tag), "_", "-"))
	if key == "" {
		return Language{}, false
	}
	i, ok := languageIndex[key]
	if !ok {
		return Language{}, false
	}
	return knownLanguages[i], true
}

// NormalizeLanguage returns the canonical tag for tag ("EN_gb" -> "en-GB",
// "en" -> "en-US", "zh-TW" -> "zh-Hant") and whether tag is known. Bare
// subtags resolve to a documented default: en -> en-US, es -> es-ES,
// pt -> pt-BR, fr -> fr-FR, de -> de-DE, zh -> zh-Hans, no/nb -> nb-NO;
// any other bare subtag resolves to its single known variant (ja -> ja-JP).
func NormalizeLanguage(tag string) (string, bool) {
	l, ok := LookupLanguage(tag)
	if !ok {
		return tag, false
	}
	return l.Tag, true
}

// DefaultLanguage is the prose language a repository has when its config
// does not name one: [DefaultConfig] sets it, [LoadConfig] fills a missing
// or empty language key with it, and init states it in every scaffolded
// agent file. There is no opt-out; choosing another language means setting
// [Config.Language].
const DefaultLanguage = "en-US"

// effectiveLanguage returns the prose language tag that applies for tag: an
// empty tag is [DefaultLanguage], a known one its canonical form, and an
// unknown one is returned unchanged (it states no directive and is reported
// by [ValidateConfig]; it is never replaced by the default behind the
// user's back).
func effectiveLanguage(tag string) string {
	if strings.TrimSpace(tag) == "" {
		return DefaultLanguage
	}
	if l, ok := LookupLanguage(tag); ok {
		return l.Tag
	}
	return tag
}

// languageDirective returns the agent directive line for tag: the
// [DefaultLanguage] directive when tag is empty, and "" when tag is unknown
// (unknown tags are reported by [ValidateConfig]).
func languageDirective(tag string) string {
	l, ok := LookupLanguage(effectiveLanguage(tag))
	if !ok {
		return ""
	}
	return l.Directive()
}

// CheckAgentsMdLanguage reports whether the root AGENTS.md states the prose
// language lang (a [Config.Language] value; empty means [DefaultLanguage]).
// It returns nothing when lang is unknown (reported by [ValidateConfig]) or
// when AGENTS.md does not exist. Otherwise it returns one result: Pass when the
// file contains the canonical tag in parentheses, e.g. "(en-US)", as the
// [Language.Directive] line does; Warn when it does not, or when the file
// cannot be read (a failed read is never taken as a pass); Skip for a
// symlink or special file, which is never followed or opened.
func CheckAgentsMdLanguage(root, lang string) []LayoutResult {
	l, ok := LookupLanguage(effectiveLanguage(lang))
	if !ok {
		return nil
	}
	const name = "AGENTS.md"
	p := filepath.Join(root, name)
	info, err := os.Lstat(p)
	if os.IsNotExist(err) {
		return nil
	}
	warn := func(msg string) []LayoutResult {
		return []LayoutResult{{Path: name, Status: Warn, Message: msg}}
	}
	if err != nil {
		return warn(fmt.Sprintf("cannot check the prose language (%s): %v -- FIX: make AGENTS.md readable", l.Tag, err))
	}
	if !info.Mode().IsRegular() {
		return []LayoutResult{{Path: name, Status: Skip, Message: "prose language not checked: not a regular file (symlink or special file); not followed"}}
	}
	f, err := os.Open(p)
	if err != nil {
		return warn(fmt.Sprintf("cannot check the prose language (%s): %v -- FIX: make AGENTS.md readable", l.Tag, err))
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxAgentsMdBytes))
	if err != nil {
		return warn(fmt.Sprintf("cannot check the prose language (%s): %v -- FIX: make AGENTS.md readable", l.Tag, err))
	}
	if !bytes.Contains(data, []byte("("+l.Tag+")")) {
		return warn(fmt.Sprintf("does not state the prose language (%s) -- FIX: wrap its Context section in %s / %s and re-run 'repogov init', or state the language in AGENTS.md by hand", l.Tag, agentsMdContextBegin, agentsMdContextEnd))
	}
	return []LayoutResult{{Path: name, Status: Pass, Message: "states the prose language (" + l.Tag + ")"}}
}

// closestLanguages returns the known tags sharing tag's primary subtag, in
// sorted order, for "did you mean" hints.
func closestLanguages(tag string) []string {
	p := primarySubtag(strings.TrimSpace(tag))
	var out []string
	for _, l := range knownLanguages {
		if primarySubtag(l.Tag) == p {
			out = append(out, l.Tag)
		}
	}
	sort.Strings(out)
	return out
}
