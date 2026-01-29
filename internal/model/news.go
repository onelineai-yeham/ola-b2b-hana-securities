package model

import "time"

// NewsSource represents the source of the news (internal)
type NewsSource string

const (
	SourceJPMinkabu NewsSource = "jp_minkabu"
	SourceCNWind    NewsSource = "cn_wind"
)

// CountryCode represents the country code for API filtering
type CountryCode string

const (
	CountryJP CountryCode = "JP"
	CountryCN CountryCode = "CN"
)

// ToNewsSource converts country code to internal news source
func (c CountryCode) ToNewsSource() NewsSource {
	switch c {
	case CountryJP:
		return SourceJPMinkabu
	case CountryCN:
		return SourceCNWind
	default:
		return ""
	}
}

// TranslatedNews represents a unified translated news record (Gold schema)
type TranslatedNews struct {
	ID                 int64      `json:"id"`
	Source             NewsSource `json:"source"`
	SourceNewsID       string     `json:"source_news_id"`
	OriginalHeadline   string     `json:"original_headline"`
	OriginalContent    *string    `json:"original_content,omitempty"`
	TranslatedHeadline string     `json:"translated_headline"`
	TranslatedContent  *string    `json:"translated_content,omitempty"`
	Tickers            []string   `json:"tickers"`
	Topics             []string   `json:"topics,omitempty"`
	Keywords           []string   `json:"keywords,omitempty"`
	Provider           *string    `json:"provider,omitempty"`
	PublishedAt        time.Time  `json:"published_at"`
	ModelName          string     `json:"model_name"`
	SourceCreatedAt    *time.Time `json:"source_created_at,omitempty"`
	SourceUpdatedAt    *time.Time `json:"source_updated_at,omitempty"`
	SyncedAt           *time.Time `json:"synced_at,omitempty"`
}

// JPMinkabuNews represents Japanese Minkabu translated news (Silver schema)
type JPMinkabuNews struct {
	ID                 int64     `json:"id"`
	NewsID             string    `json:"news_id"`
	OriginalHeadline   string    `json:"original_headline"`
	OriginalStory      *string   `json:"original_story,omitempty"`
	TranslatedHeadline string    `json:"translated_headline"`
	TranslatedStory    *string   `json:"translated_story,omitempty"`
	Providers          []string  `json:"providers"`
	Topics             []string  `json:"topics"`
	Tickers            []string  `json:"tickers"`
	CreationTime       time.Time `json:"creation_time"`
	ModelName          string    `json:"model_name"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// CNWindNews represents Chinese Wind translated news (Silver schema)
type CNWindNews struct {
	ID                int64     `json:"id"`
	ObjectID          string    `json:"object_id"`
	OriginalTitle     string    `json:"original_title"`
	OriginalContent   *string   `json:"original_content,omitempty"`
	TranslatedTitle   string    `json:"translated_title"`
	TranslatedContent *string   `json:"translated_content,omitempty"`
	PublishDate       time.Time `json:"publish_date"`
	Source            *string   `json:"source,omitempty"`
	Sections          []string  `json:"sections"`
	WindCodes         []string  `json:"wind_codes"`
	Keywords          []string  `json:"keywords"`
	ModelName         string    `json:"model_name"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ToTranslatedNews converts JPMinkabuNews to unified TranslatedNews
func (n *JPMinkabuNews) ToTranslatedNews() *TranslatedNews {
	var provider *string
	if len(n.Providers) > 0 {
		provider = &n.Providers[0]
	}
	return &TranslatedNews{
		Source:             SourceJPMinkabu,
		SourceNewsID:       n.NewsID,
		OriginalHeadline:   n.OriginalHeadline,
		OriginalContent:    n.OriginalStory,
		TranslatedHeadline: n.TranslatedHeadline,
		TranslatedContent:  n.TranslatedStory,
		Tickers:            n.Tickers,
		Topics:             n.Topics,
		Keywords:           nil,
		Provider:           provider,
		PublishedAt:        n.CreationTime,
		ModelName:          n.ModelName,
		SourceCreatedAt:    &n.CreatedAt,
		SourceUpdatedAt:    &n.UpdatedAt,
	}
}

// ToTranslatedNews converts CNWindNews to unified TranslatedNews
func (n *CNWindNews) ToTranslatedNews() *TranslatedNews {
	return &TranslatedNews{
		Source:             SourceCNWind,
		SourceNewsID:       n.ObjectID,
		OriginalHeadline:   n.OriginalTitle,
		OriginalContent:    n.OriginalContent,
		TranslatedHeadline: n.TranslatedTitle,
		TranslatedContent:  n.TranslatedContent,
		Tickers:            n.WindCodes,
		Topics:             n.Sections,
		Keywords:           n.Keywords,
		Provider:           n.Source,
		PublishedAt:        n.PublishDate,
		ModelName:          n.ModelName,
		SourceCreatedAt:    &n.CreatedAt,
		SourceUpdatedAt:    &n.UpdatedAt,
	}
}

// NewsListItem is a unified news item for API response
type NewsListItem struct {
	Date      string  `json:"date"`
	Time      string  `json:"time"`
	Publisher *string `json:"publisher,omitempty"`
	Headline  string  `json:"headline"`
	Content   *string `json:"content,omitempty"`
}

// NewsDetail is a unified detailed news for API response
type NewsDetail struct {
	ID                 string     `json:"id"`
	Source             NewsSource `json:"source"`
	OriginalHeadline   string     `json:"original_headline"`
	OriginalContent    *string    `json:"original_content,omitempty"`
	TranslatedHeadline string     `json:"translated_headline"`
	TranslatedContent  *string    `json:"translated_content,omitempty"`
	Tickers            []string   `json:"tickers"`
	Topics             []string   `json:"topics,omitempty"`
	Keywords           []string   `json:"keywords,omitempty"`
	PublishedAt        time.Time  `json:"published_at"`
	Provider           *string    `json:"provider,omitempty"`
	ModelName          string     `json:"model_name"`
}

// NewsFilter represents query parameters for news listing
type NewsFilter struct {
	Source *NewsSource
	Ticker *string
	From   *time.Time
	To     *time.Time
	Page   int
	Limit  int
}

// Pagination represents pagination info in response
type Pagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

// NewsListResponse is the API response for news listing
type NewsListResponse struct {
	Data       []NewsListItem `json:"data"`
	Pagination Pagination     `json:"pagination"`
}
