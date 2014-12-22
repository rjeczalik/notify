notify [![Build Status](https://img.shields.io/travis/yaey8Hee/Oong7Pha/wip.svg)](https://travis-ci.org/yaey8Hee/Oong7Pha "linux_amd64") [![Build Status](https://img.shields.io/travis/yaey8Hee/Oong7Pha/osx.svg)](https://travis-ci.org/yaey8Hee/Oong7Pha "darwin_amd64") [![Build status](https://img.shields.io/appveyor/ci/yaey8Hee/Oong7Pha.svg)](https://ci.appveyor.com/project/yaey8Hee/Oong7Pha "windows_amd64") [![Coverage Status](https://img.shields.io/coveralls/yaey8Hee/Oong7Pha/wip.svg)](https://coveralls.io/r/yaey8Hee/Oong7Pha?branch=wip)
======

### How it works

In theory:

![how it works](https://s3.amazonaws.com/uploads.hipchat.com/174457/1258727/b50DEkIklWrikUs/upload.png "how it works")
![how it works](https://s3.amazonaws.com/uploads.hipchat.com/174457/1258727/dZkIOdveT0sTN21/upload.png "how it works")

In practice:

![how it works](https://i.imgur.com/KZbSV.gif "how it works")

### Windows

Actions: **ADD** - `FILE_ACTION_ADDED`, **RM**   - `FILE_ACTION_REMOVED`, **CHMOD** - `FILE_ACTION_MODIFIED`, **MVOLD** - `FILE_ACTION_RENAMED_OLD_NAME`, **MVNEW** - `FILE_ACTION_RENAMED_NEW_NAME`.

Event | ADD | RM | CHMOD | MVOLD | MVNEW 
------------ | :-------------: | :------------: | :------------: | :-------------: | :------------: 
FILE_NOTIFY_CHANGE_FILE_NAME |  :page_facing_up: |  :page_facing_up: |  |  :page_facing_up: |  :page_facing_up: 
FILE_NOTIFY_CHANGE_DIR_NAME  |  :file_folder: |  :file_folder: |  |  :file_folder: |  :file_folder: 
FILE_NOTIFY_CHANGE_ATTRIBUTES  |  |  |  :page_facing_up:  :file_folder: |  |
FILE_NOTIFY_CHANGE_SIZE  |  |  |  :page_facing_up: |  |
FILE_NOTIFY_CHANGE_LAST_WRITE  |  |  |  :page_facing_up: :file_folder: |  |
FILE_NOTIFY_CHANGE_LAST_ACCESS  |  |  | :page_facing_up: :file_folder: |  |
FILE_NOTIFY_CHANGE_CREATION  |  |  |  :page_facing_up: :file_folder: |  |
FILE_NOTIFY_CHANGE_SECURITY  |  |  |  :page_facing_up: :file_folder: |  |

:page_facing_up: - file; :file_folder: - folder.