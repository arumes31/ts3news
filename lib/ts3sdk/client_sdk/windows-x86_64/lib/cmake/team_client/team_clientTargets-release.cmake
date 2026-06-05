#----------------------------------------------------------------
# Generated CMake target import file for configuration "Release".
#----------------------------------------------------------------

# Commands may need to know the format version.
set(CMAKE_IMPORT_FILE_VERSION 1)

# Import target "teamspeak::client" for configuration "Release"
set_property(TARGET teamspeak::client APPEND PROPERTY IMPORTED_CONFIGURATIONS RELEASE)
set_target_properties(teamspeak::client PROPERTIES
  IMPORTED_IMPLIB_RELEASE "${_IMPORT_PREFIX}/lib/teamspeak_sdk_client.lib"
  IMPORTED_LOCATION_RELEASE "${_IMPORT_PREFIX}/bin/teamspeak_sdk_client.dll"
  )

list(APPEND _cmake_import_check_targets teamspeak::client )
list(APPEND _cmake_import_check_files_for_teamspeak::client "${_IMPORT_PREFIX}/lib/teamspeak_sdk_client.lib" "${_IMPORT_PREFIX}/bin/teamspeak_sdk_client.dll" )

# Commands beyond this point should not need to know the version.
set(CMAKE_IMPORT_FILE_VERSION)
