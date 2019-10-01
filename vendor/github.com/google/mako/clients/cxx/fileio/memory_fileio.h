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
#ifndef CLIENTS_CXX_FILEIO_MEMORY_FILEIO_H_
#define CLIENTS_CXX_FILEIO_MEMORY_FILEIO_H_

#include <stddef.h>

#include <memory>
#include <string>
#include <vector>

#include "src/google/protobuf/message.h"
#include "spec/cxx/fileio.h"
#include "absl/base/thread_annotations.h"
#include "absl/container/flat_hash_map.h"

namespace mako {
namespace memory_fileio {

// An in-memory implementation of the Mako FileIO interface.
// The in-memory file-system is shared between instances of this class (eg. one
// instance can "see" what the others have written).
//
// Calling Clear() will clear memory of all files, and individual files can be
// cleared from memory with Delete().
//
// THIS CLASS IS NOT THREAD SAFE
// (But using different instances from threads IS safe)
class FileIO : public mako::FileIO {
 public:
  FileIO();

  // Open opens the given file path.
  // See interface docs for more information.
  bool Open(const std::string& path, mako::FileIO::AccessMode mode) override;

  // Write appends the given record to the opened file.
  // See interface docs for more information.
  bool Write(const google::protobuf::Message& record) override;
  bool Write(const std::string& serialized_record) override;

  // Read reads the next record in the opened file
  // See interface docs for more information.
  bool Read(google::protobuf::Message* record) override;
  bool Read(std::string* serialized_record) override;

  // Returns true if last call to Read() returned false and reached EOF.
  // See interface docs for more information.
  bool ReadEOF() override { return read_eof_; }

  // Returns the error message for the most recent failed call.
  // See interface docs for more information.
  std::string Error() override { return error_; }

  // Close closes the opened file.
  // See interface docs for more information.
  bool Close() override;

  // Delete deletes the given file.
  // See interface docs for more information.
  bool Delete(const std::string& path) override;

  // Returns a default instance.
  // See interface docs for more information.
  std::unique_ptr<mako::FileIO> MakeInstance() override;

  // Exposed for CLIF which needs a mako::memory_fileio::FileIO for the
  // extra methods we have defined here that are not on mako::FileIO.
  std::unique_ptr<FileIO> NewInstance();

  // Clears all in memory files.
  bool Clear();

  ~FileIO() override;

  typedef std::vector<std::string> File;

  // Injects errors. If these error strings are nonempty, then any calls to the
  // corresponding FileIO method will fail with this error.
  void set_open_error(const std::string& open_error) {
    open_error_ = open_error;
  }
  void set_read_error(const std::string& read_error) {
    read_error_ = read_error;
  }
  void set_write_error(const std::string& write_error) {
    write_error_ = write_error;
  }
  void set_delete_error(const std::string& delete_error) {
    delete_error_ = delete_error;
  }
  void set_close_error(const std::string& close_error) {
    close_error_ = close_error;
  }

 private:
  // set value to be returned by Error() to msg
  void SetError(const std::string& err_msg);
  // return vector for path_ or nullptr.
  File* GetFileForPath(const std::string& path);
  // error prefix useful for debugging
  std::string error_prefix_;
  // if true, at EOF
  bool read_eof_;
  // the latest error
  std::string error_;
  // Path that has been opened.
  std::string path_;
  // true if file has been opened for writing, false otherwise
  bool writing_;
  // index into vector when reading file. -1 otherwise.
  int read_idx_;

  // Injected errors
  std::string open_error_;
  std::string read_error_;
  std::string write_error_;
  std::string delete_error_;
  std::string close_error_;

  static absl::Mutex files_mu_;
  // Mapping from file path to list of records written to that path.
  static absl::flat_hash_map<std::string, std::unique_ptr<FileIO::File> >*
      files_ GUARDED_BY(files_mu_);

#ifndef SWIG
  // Not copyable.
  FileIO(const FileIO&) = delete;
  FileIO& operator=(const FileIO&) = delete;
#endif  // SWIG
};

}  // namespace memory_fileio
}  // namespace mako

#endif  // CLIENTS_CXX_FILEIO_MEMORY_FILEIO_H_
