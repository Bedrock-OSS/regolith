package regolith

// Common errors and warnings used by Regolith
const (
	errorConnector = "** Another error occured while handling the previous error **"

	// Error message to display when when expecting an empty or unexisting directory
	assertEmptyOrNewDirError = "Expected a path to an empty or unexisting " +
		"directory.\nPath: %s"

	// Error message for filepath.Abs() function.
	filepathAbsError = "Failed to get absolute path.\nBase path: %s"

	// Error message for os.Stat failore
	osStatErrorAny = "Failed to access file info.\nPath: %s"

	// Error message for file or directory that doesn't exist
	osStatErrorIsNotExist = "Path doesn't exist.\nPath: %s"

	// Error message for os.Stat when the funciton should fail because it's
	// expected that the target path doesn't exist
	osStatExistsError = "Path already exists.\nPath: %s"

	// Error message for handling failores of os.Rename
	osRenameError = "Failed to move file or directory:\nSource: %s\nTarget: %s"

	// Error message for handling failores of os.Copy
	osCopyError = "Failed to copy file or directory:\nSource: %s\nTarget: %s"

	// Error message displayed when mkdir (or similar function) fails
	osMkdirError = "Failed to create directory.\nPath: %s"

	// Error message displayed when os.Getwd fails
	osGetwdError = "Failed to get current working directory."

	// Common Error message to be reused on top of IsDirEmpty
	isDirEmptyError = "Failed to check if path is an empty directory.\nPath: %s"

	// Error used when an empty directory is expected but it's not
	isDirEmptyNotEmptyError = "Path is an empty directory.\nPath: %s"

	// Error used when copyFileSecurityInfo fails
	copyFileSecurityInfoError = "Failed to copy ACL.\nSource: %s\nTarget: %s"

	// Error used when RevertableFsOperations.Delete fails
	revertableFsOperationsDeleteError = "Failed to perform revertable " +
		"deletion of the file or directory.\nPath: %s"

	// Error used when filepath.Rel fails
	filepathRelError = "Failed to get relative path.\nBase: %s\nTarget: %s"

	// Error used when os.Remove (or similar function) fails
	osRemoveError = "Failed to delete file or directory.\nPath: %s"

	// Error used when MoveOrCopy function fails
	moveOrCopyError = "Failed to move or copy file or directory.\nSource: %s\nTarget: %s"

	// Error used when expecting a directory but it's not
	isDirNotADirError = "Path is not a directory.\nPath: %s"

	// Error used when os.Open fails
	osOpenError = "Failed to open.\nPath: %s"

	// Error used when os.Create fails
	osCreateError = "Failed to open for writing.\nPath: %s"

	// Error used when os.Rel fails
	osRelError = "Failed to get relative path.\nBase: %s\nTarget: %s"

	// Error used when os.Walk fails
	osWalkError = "Failed to walk directory.\nPath: %s"

	// Error used when program fails to read from opened file
	fileReadError = "Failed to read from file.\nPath: %s"

	// Error used when program fails to write to opened file
	fileWriteError = "Failed to write to file.\nPath: %s"

	// Error used when program fails to parse JSON file
	jsonUnmarshalError = "Failed to parse JSON.\nPath: %s"

	// Error used when Regolith fails to parse a property os JSON
	jsonPropertyParseError = "Failed to parse JSON property.\nProperty: %s"

	// Error used when Regolith expects a property but it's missing
	jsonPropertyMissingError = "Required JSON property is missing.\nProperty: %s"

	// Errror used when JSON property is not an expected type
	jsonPropertyTypeError = "JSON property has unexpected type." +
		"\nProperty: %s\nExpected: %s"

	// Error used when Regolith fails to parse a property os JSON
	jsonPathParseError = "Failed to parse JSON.\nJSON Path: %s"

	// Error used when JSON path is missing
	jsonPathMissingError = "Required JSON path is missing.\nJSON Path: %s"

	// Error used when JSON path exists but the type is wrong
	jsonPathTypeError = "Invalid data type.\nJSON Path: %s\n" +
		"Expected type: %s"

	// Error used when RunSubProcess funciton fails
	runSubProcessError = "Failed to run sub process."

	// Error used when remote filter fails to access its subfilter collection.
	// The error doesn't print the name of the filter because the
	// subfilterCollection method is private and it's always a part of some
	// other, higher level action which provides that information when it
	// fails.
	remoteFilterSubfilterCollectionError = "Failed to list subfilters."

	// Error used when GetRemoteFilterDownloadRef function fails
	getRemoteFilterDownloadRefError = "Failed to get download link for the filter.\n" +
		"Filter repository Url: %s\n" +
		"Filter name: %s\n" +
		"Filter version: %s"

	// Error used when CreateFilterRunner method of FilterInstaller fails
	createFilterRunnerError = "Failed to create filter runner.\nFilter: %s"

	// Warning used when Git is not installed
	gitNotInstalledWarning = "Git is not installed. Git is required to download " +
		"filters.\n You can download Git from https://git-scm.com/downloads"

	// Error used when filterFromObject function fails
	filterFromObjectError = "Failed to parse filter from JSON object."

	// Error used when remote filter fails to download
	remoteFilterDownloadError = "Failed to download filter.\nFilter: %s"

	// Error used when exec.Command fails.
	execCommandError = "Failed to execute command.\nCommand: %s"

	// Error used when FilterRunner.Check method fails
	filterRunnerCheckError = "Filter check failed.\nFilter: %s"

	// Error used when certain function is not implemented on this system
	notImplementedOnThisSystemError = "Not implemented for this system."

	// Error used when SetupTmpFiles function fails
	setupTmpFilesError = "Failed to setup temporary files.\n" +
		"Regolith files path: %s" // .regolith

	// Error used when ExportProject function fails
	exportProjectError = "Failed to export project."

	// Error used when RunContext.GetProfile function fails
	runContextGetProfileError = "Failed to get profile."

	filterRunnerRunError = "Failed to run filter.\nFilter: %s"

	// Error used when GetRegolithConfigPath fails
	getRegolithConfigPathError = "Failed to get path to Regolith's app data folder."

	// Error used whe Regolith fails to undo failed file system operation.
	fsUndoError = "Filed to undo file system operation."

	// Error used when aquireSessionLock function fails
	aquireSessionLockError = "Failed to aquire session lock."
)
