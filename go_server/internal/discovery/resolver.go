package discovery

import (
	"context"
	"errors"
	"sort"
	"strings"

	"comics/domain"
	"comics/internal/identity"
)

type MatchKind string

const (
	MatchNone         MatchKind = ""
	MatchDiscoveryKey MatchKind = "discovery_key"
	MatchTitlePrefix  MatchKind = "title_prefix"
	MatchTitleTokens  MatchKind = "title_tokens"
)

const (
	scoreExactText       = 15
	scoreExactKey        = 10
	scoreTitleTokens     = 8
	scoreTitlePrefix     = 6
	minimumAcceptedScore = 6
	minimumWinningMargin = 2
)

type ResolveOptions struct {
	ExcludeComicID int
}

type Resolver struct {
	observer Observer
}

func NewResolver(observer Observer) *Resolver {
	if observer == nil {
		observer = NoopObserver{}
	}
	return &Resolver{observer: observer}
}

var resolveComicResolver = NewResolver(DefaultObserver())

// ResolveComic finds an existing comic for a scraped or created title.
func ResolveComic(
	ctx context.Context,
	repo domain.ComicRepository,
	incomingTitle string,
	comType int,
) (domain.Comic, MatchKind, bool, error) {
	return resolveComicResolver.Resolve(ctx, repo, incomingTitle, comType, ResolveOptions{})
}

func (r *Resolver) Resolve(
	ctx context.Context,
	repo domain.ComicRepository,
	incomingTitle string,
	comType int,
	options ResolveOptions,
) (domain.Comic, MatchKind, bool, error) {
	titles := identity.NormalizeTitleVariants([]string{incomingTitle}, comType)
	if len(titles) == 0 {
		r.observer.ObserveResolution(OutcomeNotFound, MatchNone)
		return domain.Comic{}, MatchNone, false, nil
	}
	primary := titles[0]
	scope := identity.DiscoveryScope(primary, comType)
	titleKeys := identity.DiscoveryTitleKeysForTitles(titles, comType)

	matches, err := repo.FindComicsByDiscoveryTitleKeys(ctx, scope, titleKeys)
	if err != nil {
		r.observer.ObserveResolution(OutcomeError, MatchNone)
		return domain.Comic{}, MatchNone, false, err
	}
	matches = excludeComicID(matches, options.ExcludeComicID)
	matches = filterByTitleRelation(matches, primary, comType)
	comic, ok, err := chooseCandidate(primary, scope, MatchDiscoveryKey, matches, comType)
	if err != nil {
		var ambiguous *domain.AmbiguousComicMatchError
		if errors.As(err, &ambiguous) {
			r.observer.ObserveResolution(OutcomeAmbiguous, MatchDiscoveryKey)
		} else {
			r.observer.ObserveResolution(OutcomeError, MatchDiscoveryKey)
		}
		return domain.Comic{}, MatchDiscoveryKey, false, err
	}
	if ok {
		r.observer.ObserveResolution(OutcomeMatched, MatchDiscoveryKey)
		return comic, MatchDiscoveryKey, true, nil
	}

	token := titleLookupSearchTerm(primary)
	if token != "" {
		candidates, err := repo.FindComicsByDiscoveryKeyToken(ctx, scope, token, 200)
		if err != nil {
			r.observer.ObserveResolution(OutcomeError, MatchNone)
			return domain.Comic{}, MatchNone, false, err
		}
		candidates = excludeComicID(candidates, options.ExcludeComicID)
		candidates = filterByTitleRelation(candidates, primary, comType)
		kind := MatchTitlePrefix
		if len(candidates) > 0 && matchUsesTokens(primary, candidates[0]) {
			kind = MatchTitleTokens
		}
		comic, ok, err = chooseCandidate(primary, scope, kind, candidates, comType)
		if err != nil {
			var ambiguous *domain.AmbiguousComicMatchError
			if errors.As(err, &ambiguous) {
				r.observer.ObserveResolution(OutcomeAmbiguous, kind)
			} else {
				r.observer.ObserveResolution(OutcomeError, kind)
			}
			return domain.Comic{}, kind, false, err
		}
		if ok {
			r.observer.ObserveResolution(OutcomeMatched, kind)
			return comic, kind, true, nil
		}
	}

	r.observer.ObserveResolution(OutcomeNotFound, MatchNone)
	return domain.Comic{}, MatchNone, false, nil
}

type scoredCandidate struct {
	Comic domain.Comic
	Score int
}

func excludeComicID(comics []domain.Comic, excludeID int) []domain.Comic {
	if excludeID == 0 {
		return comics
	}
	out := make([]domain.Comic, 0, len(comics))
	for _, comic := range comics {
		if comic.ID != excludeID {
			out = append(out, comic)
		}
	}
	return out
}

func scoreCandidates(candidates []domain.Comic, incomingPrimary string, comType int) []scoredCandidate {
	out := make([]scoredCandidate, 0, len(candidates))
	for _, comic := range candidates {
		if !comicInLookupScope(comic, comType) {
			continue
		}
		score := relationshipScore(incomingPrimary, comic)
		if score > 0 {
			out = append(out, scoredCandidate{Comic: comic, Score: score})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].Comic.ID < out[j].Comic.ID
	})
	return out
}

func relationshipScore(incomingPrimary string, comic domain.Comic) int {
	incomingKey := identity.TitleMatchKey(incomingPrimary)
	best := 0
	for _, stored := range comic.Titles {
		score := 0
		if strings.EqualFold(strings.TrimSpace(stored), strings.TrimSpace(incomingPrimary)) {
			score = scoreExactText
		} else if identity.TitleMatchKey(stored) == incomingKey && incomingKey != "" {
			score = scoreExactKey
		} else if identity.TitlesMatchBySignificantTokens(incomingPrimary, stored) {
			score = scoreTitleTokens
		} else if identity.TitlesArePrefixMatch(incomingPrimary, stored) {
			score = scoreTitlePrefix
		}
		if score > best {
			best = score
		}
	}
	return best
}

func chooseCandidate(
	incoming string,
	scope string,
	kind MatchKind,
	candidates []domain.Comic,
	comType int,
) (domain.Comic, bool, error) {
	scored := scoreCandidates(candidates, incoming, comType)
	if len(scored) == 0 || scored[0].Score < minimumAcceptedScore {
		return domain.Comic{}, false, nil
	}
	if len(scored) > 1 && scored[0].Score-scored[1].Score < minimumWinningMargin {
		ids := []int{scored[0].Comic.ID, scored[1].Comic.ID}
		for _, item := range scored[2:] {
			if scored[0].Score-item.Score >= minimumWinningMargin {
				break
			}
			ids = append(ids, item.Comic.ID)
		}
		sort.Ints(ids)
		return domain.Comic{}, false, &domain.AmbiguousComicMatchError{
			IncomingTitle: incoming,
			Scope:         scope,
			MatchKind:     string(kind),
			CandidateIDs:  ids,
			BestScore:     scored[0].Score,
			RunnerUpScore: scored[1].Score,
		}
	}
	return scored[0].Comic, true, nil
}

func matchUsesTokens(primary string, comic domain.Comic) bool {
	for _, stored := range comic.Titles {
		if identity.TitlesMatchBySignificantTokens(primary, stored) {
			return true
		}
	}
	return false
}

var titleLookupStopWords = map[string]struct{}{
	"became": {}, "become": {}, "being": {}, "return": {}, "returns": {},
	"from": {}, "with": {}, "into": {}, "that": {}, "this": {},
}

func filterByTitleRelation(candidates []domain.Comic, primary string, comType int) []domain.Comic {
	out := make([]domain.Comic, 0, len(candidates))
	for _, comic := range candidates {
		if !comicInLookupScope(comic, comType) {
			continue
		}
		for _, stored := range comic.Titles {
			if identity.TitleMatchKey(stored) == identity.TitleMatchKey(primary) ||
				identity.TitlesArePrefixMatch(primary, stored) ||
				identity.TitlesMatchBySignificantTokens(primary, stored) {
				out = append(out, comic)
				break
			}
		}
	}
	return out
}

func comicInLookupScope(comic domain.Comic, incomingComType int) bool {
	comicIsNovel := comic.ComType == identity.NovelType
	for _, stored := range comic.Titles {
		if identity.IsNovelIdentity(stored, comic.ComType) {
			comicIsNovel = true
			break
		}
	}
	incomingIsNovel := incomingComType == identity.NovelType
	return comicIsNovel == incomingIsNovel
}

func titleLookupSearchTerm(primaryTitle string) string {
	words := strings.Fields(strings.TrimSpace(primaryTitle))
	if len(words) == 0 {
		return ""
	}
	var candidates []string
	for _, word := range words {
		lower := strings.ToLower(word)
		if len(word) < 4 {
			continue
		}
		if lower == "the" || lower == "a" || lower == "an" {
			continue
		}
		if _, stop := titleLookupStopWords[lower]; stop {
			continue
		}
		candidates = append(candidates, lower)
	}
	if len(candidates) == 0 {
		for _, word := range words {
			lower := strings.ToLower(word)
			if lower != "the" && lower != "a" && lower != "an" && word != "" {
				return lower
			}
		}
		return strings.ToLower(words[0])
	}
	best := candidates[0]
	for _, word := range candidates[1:] {
		if len(word) > len(best) {
			best = word
		}
	}
	return best
}
