// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// see the license for the specific language governing permissions and
// limitations under the license.
#ifndef SPEC_CXX_FILEIO_H_
#define SPEC_CXX_FILEIO_H_

#include <stddef.h>
#include <memory>
#include <string>

#include "src/google/protobuf/message.h"

namespace mako {

// FileIO is the interface for opening, reading, writing, and deleting
// files of many protobuf records.
//
// Users pass an implementation to the load framework. The framework may create
// other instances with calls to MakeInstance().
//
// Users creating data queues should use the same FileIO impl as the
// target Load component when creating the files.
class FileIO {
 public:
  enum AccessMode {
    // If file does not exist, it is created. If the file does exist, contents
    // are overwritten.
    //
    // When opened in this mode, the FileIO instance is writable, allowing
    // only Write and WriteBytes calls.
    kWrite,
    // Open for reading.
    //
    // When opened in this mode, the FileIO instance is readable, allowing
    // only Read and ReadBytes calls.
    kRead,
    // If file does not exist, it is created. If the file does exist, new
    // writes are appended to the end.
    //
    // When opened in this mode, the FileIO instance is writable, allowing
    // only Write and WriteBytes calls.
    kAppend,
  };

  // Open opens the given file path.
  //
  // When opening a file in kWrite or kAppend mode, any directories in the path
  // that do not exist must also be created.
  //
  // This instance must be unopened when calling Open().
  //
  // Returns true for success. If false is returned, call Error() to get the
  // error message.
  virtual bool Open(const std::string& path, AccessMode mode) = 0;

  // Write appends the given record to the opened file.
  //
  // Notes:
  // * This instance must be writable when calling Write().
  // * Flushes happen at the timing and discretion of the implementation, and at
  // Close().
  //
  // Returns true for success. If false is returned, call Error() to get the
  // error message.
  virtual bool Write(const google::protobuf::Message& record) = 0;

  // Overload exposed for CLIF and SWIG.
  virtual bool Write(const std::string& serialized_record) = 0;

  // Read reads the next record in the opened file.
  //
  // This instance must be readable mode when calling Read().
  //
  // Returns true for success.
  // If false is returned, it could be due to EOF or error:
  //   * Call ReadEOF() to check if reached EOF.
  //   * If not EOF, Call Error() to get the error message.
  // Example:
  //   bool success = true;
  //   while (success) {
  //     success = rrw.Read(...);
  //   }
  //   if (!rrw.ReadEOF()) {
  //     log << rrw.Error();
  //     // handle error
  //   }
  virtual bool Read(google::protobuf::Message* record) = 0;

  // Overload exposed for CLIF and SWIG.
  virtual bool Read(std::string* serialized_record) = 0;

  // Returns true if last call to Read() returned false and reached EOF.
  virtual bool ReadEOF() = 0;

  // Returns the error message for the most recent failed call.
  virtual std::string Error() = 0;

  // Close closes the opened file.
  //
  // Notes:
  // * Calling Close() on an unopened instance is a no-op.
  // * Close() must be called when done reading/writing a file.
  // * When writable, the file will be flushed before closing.
  //
  // Returns true for success. If false is returned, call Error() to get the
  // error message.
  virtual bool Close() = 0;

  // Delete deletes the given file.
  //
  // This instance must not be opened when calling Delete().
  //
  // Returns true for success. If false is returned, call Error() to get the
  // error message.
  virtual bool Delete(const std::string& path) = 0;

  // Makes another default instance of this FileIO().
  //
  // NOTE: This is not the current instance, it is used by the framework
  // to create many instances.
  virtual std::unique_ptr<FileIO> MakeInstance() = 0;

  // Detructor should call Close() if the user has not called it,
  // and ignore any error that may be returned.
  virtual ~FileIO() {}
};

}  // namespace mako

#endif  // SPEC_CXX_FILEIO_H_
