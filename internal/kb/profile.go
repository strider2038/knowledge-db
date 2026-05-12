package kb

type SourceKind string

const (
	SourceKindRepository       SourceKind = "repository"
	SourceKindDocumentation    SourceKind = "documentation"
	SourceKindProductService   SourceKind = "product_service"
	SourceKindOnlineTool       SourceKind = "online_tool"
	SourceKindDirectoryCatalog SourceKind = "directory_catalog"
	SourceKindLearningResource SourceKind = "learning_resource"
	SourceKindArticle          SourceKind = "article"
	SourceKindNews             SourceKind = "news"
	SourceKindSocialPost       SourceKind = "social_post"
	SourceKindUnknown          SourceKind = "unknown"
)

type ContentProfile string

const (
	ContentProfileRepository       ContentProfile = "repository_profile"
	ContentProfileProduct          ContentProfile = "product_profile"
	ContentProfileDocumentation    ContentProfile = "documentation_profile"
	ContentProfileOnlineTool       ContentProfile = "online_tool_profile"
	ContentProfileDirectory        ContentProfile = "directory_profile"
	ContentProfileLearningResource ContentProfile = "learning_resource_profile"
	ContentProfileConceptualDigest ContentProfile = "conceptual_digest"
	ContentProfileBriefDigest      ContentProfile = "brief_digest"
	ContentProfileLinkBookmark     ContentProfile = "link_bookmark"
)

func IsValidSourceKind(value string) bool {
	switch SourceKind(value) {
	case SourceKindRepository,
		SourceKindDocumentation,
		SourceKindProductService,
		SourceKindOnlineTool,
		SourceKindDirectoryCatalog,
		SourceKindLearningResource,
		SourceKindArticle,
		SourceKindNews,
		SourceKindSocialPost,
		SourceKindUnknown:
		return true
	default:
		return false
	}
}

func IsValidContentProfile(value string) bool {
	switch ContentProfile(value) {
	case ContentProfileRepository,
		ContentProfileProduct,
		ContentProfileDocumentation,
		ContentProfileOnlineTool,
		ContentProfileDirectory,
		ContentProfileLearningResource,
		ContentProfileConceptualDigest,
		ContentProfileBriefDigest,
		ContentProfileLinkBookmark:
		return true
	default:
		return false
	}
}
