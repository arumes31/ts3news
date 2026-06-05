#----------------------------------------------------------------
# Generated CMake target import file for configuration "Release".
#----------------------------------------------------------------

# Commands may need to know the format version.
set(CMAKE_IMPORT_FILE_VERSION 1)

# Import target "teamspeak::server" for configuration "Release"
set_property(TARGET teamspeak::server APPEND PROPERTY IMPORTED_CONFIGURATIONS RELEASE)
set_target_properties(teamspeak::server PROPERTIES
  IMPORTED_LOCATION_RELEASE "${_IMPORT_PREFIX}/lib/libteamspeak_sdk_server.so"
  IMPORTED_SONAME_RELEASE "libteamspeak_sdk_server.so"
  )

list(APPEND _cmake_import_check_targets teamspeak::server )
list(APPEND _cmake_import_check_files_for_teamspeak::server "${_IMPORT_PREFIX}/lib/libteamspeak_sdk_server.so" )

# Commands beyond this point should not need to know the version.
set(CMAKE_IMPORT_FILE_VERSION)
