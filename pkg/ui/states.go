package ui

// current view/page
type State int

const (
	StateDefault  State = iota // landing / welcome screen
	StateHome                  // home / bio
	StateProjects              // projects list + detail
	StateBlog                  // blog pointer
	StateContact               // contact info
	StateMessages              // leave-a-message form
)
