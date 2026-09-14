package agent

func searchContentNote(config Config) string {
	if config.SearchContent == nil || !*config.SearchContent {
		return ""
	}
	return "\nsearch_content searches file contents with a regular expression, returning matching lines with their file and line number. Narrow with a glob when you know the file type. Use it instead of reading files one at a time to find where something is defined or used.\n"
}

func findFilesNote(config Config) string {
	if config.FindFiles == nil || !*config.FindFiles {
		return ""
	}
	return "\nfind_files lists files matching a glob, relative to the working directory. Use it to learn what exists before reading anything. Generated directories such as .git, node_modules and vendor are skipped.\n"
}
