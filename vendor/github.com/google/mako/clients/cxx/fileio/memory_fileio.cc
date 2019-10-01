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
#include "clients/cxx/fileio/memory_fileio.h"

#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "src/google/protobuf/message.h"
#include "absl/container/flat_hash_map.h"
#include "absl/memory/memory.h"
#include "absl/strings/str_cat.h"
#include "absl/synchronization/mutex.h"

namespace mako {
namespace memory_fileio {

ABSL_CONST_INIT absl::Mutex FileIO::files_mu_(absl::kConstInit);
// Mapping from file path to list of records written to that path.
absl::flat_hash_map<std::string, std::unique_ptr<FileIO::File>>*
    FileIO::files_ =
        new absl::flat_hash_map<std::string, std::unique_ptr<FileIO::File>>;

FileIO::FileIO()
    : error_prefix_("memory_fileio::FileIO "),
      read_eof_(false),
      writing_(false),
      read_idx_(-1) {}

void FileIO::SetError(const std::string& err_msg) {
  if (err_msg.empty()) {
    error_ = err_msg;
  } else {
    error_ = absl::StrCat(error_prefix_, err_msg);
  }
}

bool FileIO::Open(const std::string& path, mako::FileIO::AccessMode mode) {
  if (!open_error_.empty()) {
    SetError(open_error_);
    return false;
  }
  if (writing_ || read_idx_ > -1) {
    SetError("File is still open for reading or writing. Call Close() first.");
    return false;
  }

  absl::MutexLock lock(&files_mu_);

  if (mode == mako::FileIO::AccessMode::kWrite) {
    (*files_)[path] = absl::make_unique<FileIO::File>();
    writing_ = true;
  } else if (mode == mako::FileIO::AccessMode::kAppend) {
    if (files_->count(path) == 0) {
      (*files_)[path] = absl::make_unique<FileIO::File>();
    }
    writing_ = true;
  } else {
    if (nullptr == GetFileForPath(path)) {
      SetError(absl::StrCat("File not found at: ", path));
      return false;
    }
    read_idx_ = 0;
  }
  // Save the path we're operating on.
  path_ = path;
  return true;
}

bool FileIO::Write(const std::string& serialized_record) {
  if (!write_error_.empty()) {
    SetError(write_error_);
    return false;
  }
  if (!writing_) {
    SetError("File has not been opened for writing.");
    return false;
  }
  absl::MutexLock lock(&files_mu_);
  auto file = GetFileForPath(path_);
  if (nullptr == file) {
    SetError("File cannot be written to. It has been deleted.");
    return false;
  }
  file->push_back(serialized_record);
  return true;
}

bool FileIO::Write(const google::protobuf::Message& record) {
  if (!write_error_.empty()) {
    SetError(write_error_);
    return false;
  }
  std::string serialized_record;
  if (record.SerializeToString(&serialized_record)) {
    return Write(serialized_record);
  }
  SetError("Failed to serialize record to string");
  return false;
}

bool FileIO::Read(std::string* serialized_record) {
  if (!read_error_.empty()) {
    SetError(read_error_);
    return false;
  }
  if (read_idx_ < 0) {
    SetError("File is not open for read.");
    return false;
  }
  absl::MutexLock lock(&files_mu_);
  auto file = GetFileForPath(path_);
  if (nullptr == file) {
    SetError("File cannot be read from. It has been deleted.");
    return false;
  }
  if (static_cast<std::size_t>(read_idx_) >=
      file->size()) {
    SetError("EOF");
    read_eof_ = true;
    return false;
  }
  *serialized_record = file->at(read_idx_++);
  return true;
}


bool FileIO::Read(google::protobuf::Message* record) {
  if (!read_error_.empty()) {
    SetError(read_error_);
    return false;
  }
  std::string serialized_record;
  if (!Read(&serialized_record)) {
    return false;
  }
  if (record->ParseFromString(serialized_record)) {
    return true;
  }
  SetError("Failed to parse record from std::string.");
  return false;
}

FileIO::File* FileIO::GetFileForPath(const std::string& path)
    EXCLUSIVE_LOCKS_REQUIRED(files_mu_) {
  auto search = files_->find(path);
  return search == files_->end() ? nullptr : search->second.get();
}

bool FileIO::Delete(const std::string& path) {
  if (!delete_error_.empty()) {
    SetError(delete_error_);
    return false;
  }
  if (writing_ || read_idx_ > -1) {
    SetError("File is still open for reading or writing. Call Close() first.");
    return false;
  }
  absl::MutexLock lock(&files_mu_);
  if (files_->erase(path) != 1) {
    SetError(absl::StrCat("No such file path: ", path));
    return false;
  }
  return true;
}

bool FileIO::Clear() {
  absl::MutexLock lock(&files_mu_);
  files_->clear();
  return true;
}

bool FileIO::Close() {
  if (!close_error_.empty()) {
    SetError(close_error_);
    return false;
  }
  // Clear out error.
  SetError("");
  path_ = "";
  writing_ = false;
  read_idx_ = -1;
  return true;
}

std::unique_ptr<mako::FileIO> FileIO::MakeInstance() {
  return NewInstance();
}

std::unique_ptr<FileIO> FileIO::NewInstance() {
  auto fileio = absl::make_unique<FileIO>();
  fileio->set_open_error(open_error_);
  fileio->set_read_error(read_error_);
  fileio->set_write_error(write_error_);
  fileio->set_delete_error(delete_error_);
  fileio->set_close_error(close_error_);
  return fileio;
}

FileIO::~FileIO() { Close(); }

}  // namespace memory_fileio
}  // namespace mako
