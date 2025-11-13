package regolith

// Common errors and warnings used by Regolith
const (
	errorConnector = "** Another error occurred while handling the previous error **"

	// Error message to display when expecting an empty or nonexistent directory
	assertEmptyOrNewDirError = "Expected a path to an empty or nonexistent " +
		"directory.\nPath: %s"

	// Error message for filepath.Abs() function.
	filepathAbsError = "Failed to get absolute path.\nBase path: %s"

	// Error message for os.ReadDir() failure
	osReadDirError = "Failed to list files in the directory.\nPath: %s"

	// Error message for os.Stat failure
	osStatErrorAny = "Failed to access file info.\nPath: %s"

	// Error message for file or directory that doesn't exist
	osStatErrorIsNotExist = "Path doesn't exist.\nPath: %s"

	// Error message for os.Stat when the function should fail because it's
	// expected that the target path doesn't exist
	osStatExistsError = "Path already exists.\nPath: %s"

	// Error message for handling failures of os.Rename
	osRenameError = "Failed to move file or directory:\nSource: %s\nTarget: %s"

	// Error message for handling failures of os.Copy
	osCopyError = "Failed to copy file or directory:\nSource: %s\nTarget: %s"

	// Error message displayed when mkdir (or similar function) fails
	osMkdirError = "Failed to create directory.\nPath: %s"

	// Error message displayed when os.Getwd fails
	osGetwdError = "Failed to get current working directory."

	// Error message displayed when os.Getwd fails
	osChtimesError = "Failed to update file modification time.\nPath: %s"

	// Common Error message to be reused on top of IsDirEmpty
	isDirEmptyError = "Failed to check if path is an empty directory.\nPath: %s"

	// Error used when an empty directory is expected, but it's not
	isDirEmptyNotEmptyError = "Path is not an empty directory.\nPath: %s"

	// Error used when copyFileSecurityInfo fails
	copyFileSecurityInfoError = "Failed to copy ACL.\nSource: %s\nTarget: %s"

	// Error used when revertibleFsOperations.Delete fails
	revertibleFsOperationsDeleteError = "Failed to perform revertible " +
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

	// Error used when Regolith expects a property, but it's missing
	jsonPropertyMissingError = "Required JSON property is missing.\nProperty: %s"

	// Error used when JSON property is not an expected type
	jsonPropertyTypeError = "JSON property has unexpected type." +
		"\nProperty: %s\nExpected: %s"

	// Error used when Regolith fails to parse a property os JSON
	jsonPathParseError = "Failed to parse JSON.\nJSON Path: %s"

	// Error used when JSON path is missing
	jsonPathMissingError = "Required JSON path is missing.\nJSON Path: %s"

	// Error used when JSON path exists but the type is wrong
	jsonPathTypeError = "Invalid data type.\nJSON Path: %s\n" +
		"Expected type: %s"

	// Error used when RunSubProcess function fails
	runSubProcessError = "Failed to run sub process."

	// Error used when remote filter fails to access its subfilter collection.
	// The error doesn't print the name of the filter because the
	// subfilterCollection method is private, and it's always a part of some
	// other, higher level action which provides that information when it
	// fails.
	remoteFilterSubfilterCollectionError = "Failed to list subfilters."

	// Error used when GetRemoteFilterDownloadRef function fails
	getRemoteFilterDownloadRefError = "Failed to get download link for the filter.\n" +
		"Filter repository Url: %s\n" +
		"Filter name: %s\n" +
		"Filter version: %s"

	// Error used when FilterDefinitionFromTheInternet function fails to handle the manifest
	filterDefinitionFromTheInternetError = "Failed to get the filter definition from the internet.\n" +
		"Filter repository Url: %s\n" +
		"Filter name: %s\n" +
		"Filter version: %s"

	// Error used when ManifestForRepo function fails to handle the manifest
	getRemoteManifestError = "Failed to get the manifest for the filter.\n" +
		"Filter repository Url: %s\n" +
		"Filter name: %s\n" +
		"Filter version: %s"

	isUrlBasedRemoteFitlerError = "Failed to determine if the filter is URL-based.\n" +
		"Filter name: %s"

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

	// Error used when env variable COM_MOJANG is not set on non Windows system
	comMojangEnvUnsetError = "COM_MOJANG environment variable is not set."

	// Error used when env variable COM_MOJANG_PREVIEW is not set on non Windows system
	comMojangPreviewEnvUnsetError = "COM_MOJANG_PREVIEW environment variable is not set."

	// Error used when env variable COM_MOJANG_EDU is not set on non Windows system
	comMojangEduEnvUnsetError = "COM_MOJANG_EDU environment variable is not set."

	// Error used when SetupTmpFiles function fails
	setupTmpFilesError = "Failed to setup temporary files.\n" +
		"Regolith files path: %s" // .regolith

	// Error used when ExportProject function fails
	exportProjectError = "Failed to export project."

	// Error used when RunContext.GetProfile function fails
	runContextGetProfileError = "Failed to get profile."

	filterRunnerRunError = "Failed to run filter.\nFilter: %s"

	// Error used when GetRegolithConfigPath fails
	getRegolithAppDataPathError = "Failed to get path to Regolith's app data folder."

	// Error used when GetUserConfig function fails
	getUserConfigError = "Failed to get user configuration."

	// Error used whe Regolith fails to undo failed file system operation.
	fsUndoError = "Filed to undo file system operation."

	// Error used when acquireSessionLock function fails
	acquireSessionLockError = "Failed to acquire session lock."

	// Error used when creation of the RevertibleFsOperations object fails
	newRevertibleFsOperationsError = "Failed to prepare backup path for revertible" +
		" file system operations.\n" +
		"Path that Regolith tried to use: %s"

	// Error used when source files (RP, BP or data) before updating them fails.
	// This should trigger an Undo() operation from the RevertibleFsOperations
	// object.
	updateSourceFilesError = "Failed to clear source files while updating them with a new version.\n" +
		"Path: %s\n" +
		"The most common reason for this problem is that the data path is used by another " +
		"program (usually terminal).\n" +
		"Please close your terminal and try again.\n" +
		"Make sure that you open it directly inside the root of the Regolith project."

	// Error used on attempt to access user config property that is not known
	// to Regolith.
	invalidUserConfigPropertyError = "Invalid user configuration property:\n" +
		"Property name: %s\n"

	// Error used when the getGlobalUserConfigPath function fails
	getGlobalUserConfigPathError = "Failed to get global user_config.json path"

	// Error used when the dump method of the UserConfig object fails
	userConfigDumpError = "Failed to save the user configuration.\n" +
		"Path: %s"

	// readFilterJsonError is used when loading the filter.json file fails
	readFilterJsonError = "Couldn't read filter data from path.\n" +
		"Path: %s\n" +
		"Did you install the filter?\n" +
		"You can install all of the filters by running:\n" +
		"regolith install-all"

	projectInMojangDirError = "Project is in the Minecraft development packs directory.\n" +
		"Path: %s\n" +
		"Minecraft directory: %s"

	projectInPreviewDirError = "Project is in the Minecraft Preview development packs directory.\n" +
		"Path: %s\n" +
		"Minecraft directory: %s"

	projectSuspiciousDirError = "Cannot initialize the project in a suspicious location."

	resolverPathCacheError = "Failed to get the cache path of the resolver file.\n" +
		"Short URL: %s"

	resolverResolveUrlError = "Failed to resolve the URL of the resolver file for the download.\n" +
		"Short URL: %s"

	resolverParseDurationError = "Failed to parse resolver cache update cooldown.\n" +
		"Cooldown specified in user settings: %s"

	// findMojangDirError is used when the FindMojangDir function fails
	findMojangDirError = "Failed to find \"com.mojang\" directory."

	// findPreviewDirError is used when the FindPreviewDir function fails
	findPreviewDirError = "Failed to find the preview \"com.mojang\" directory."

	// findEducationDirError is used when the FindEducationDir function fails
	findEducationDirError = "Failed to find the \"com.mojang\" directory."

	invalidExportPathError = "The build property of the export is invalid:\n" +
		"Current value: %q\n" +
		"Valid values are: %s"

	// getExportPathsError is used when the GetExportPaths function fails.
	getExportPathsError = "Failed to get generate export paths."

	// Error used when the formatVersion of the config file is incompatible
	// with the current version of Regolith.
	incompatibleFormatVersionError = "Incompatible formatVersion: \n" +
		"Version in config: %s\n" +
		"Latest compatible version: %s"

	// Error used when createDirLink fails
	createDirLinkError = "Failed to create directory link.\nSource: %s\nTarget: %s"

	// Error used when CheckDeletionSafety fails
	checkDeletionSafetyError = "Safety mechanism stopped Regolith to protect unexpected files " +
		"from your export targets.\n" +
		"Did you edit the exported files manually?\n" +
		"Please clear your export paths and try again.\n" +
		"Resource pack export path: %s\n" +
		"Behavior pack export path: %s"

	updatedFilesDumpError = "Failed to update the list of the files edited by Regolith." +
		"This may cause the next run to fail."

	invalidArgumentModeError = "The extraArguments property of a filter is invalid:\n" +
		"Current value: %q\n" +
		"Valid values are: %s"
)
