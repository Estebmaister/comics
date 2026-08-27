package scrape

const (
	PublisherAsura        = 1
	PublisherManhuaPlus   = 3
	PublisherFlameScans   = 4
	PublisherRealmScans   = 8
	PublisherManganato    = 17
	PublisherDemonicScans = 19
)

const (
	ComTypeUnknown = 0
	ComTypeManga   = 1
	ComTypeManhua  = 2
	ComTypeManhwa  = 3
	ComTypeNovel   = 4

	StatusUnknown   = 0
	StatusCompleted = 1
	StatusOnAir     = 2
	StatusBreak     = 3
	StatusDropped   = 4
)

var (
	coverUpdatePublishers = map[int]struct{}{
		PublisherAsura:        {},
		PublisherFlameScans:   {},
		PublisherManganato:    {},
		PublisherRealmScans:   {},
		PublisherDemonicScans: {},
	}
	restrictedCoverPublishers = map[int]struct{}{
		PublisherManhuaPlus: {},
		PublisherManganato:  {},
	}
	lowPriorityCoverPublishers = map[int]struct{}{
		PublisherManganato: {},
	}
)
