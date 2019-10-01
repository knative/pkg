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

#include "gtest/gtest.h"
#include "spec/proto/mako.pb.h"

namespace mako {
namespace memory_fileio {

class MemoryFileioTest : public ::testing::Test {
 protected:
  MemoryFileioTest() {}

  ~MemoryFileioTest() override {}

  void SetUp() override {
    FileIO f;
    f.Clear();
  }

  void TearDown() override {}
};

TEST_F(MemoryFileioTest, OpenForWritingTwice) {
  FileIO f;
  std::string file_path = "/tmp/blah";
  ASSERT_TRUE(f.Open(file_path, mako::FileIO::AccessMode::kWrite));
  ASSERT_FALSE(f.Open(file_path, mako::FileIO::AccessMode::kWrite));
  ASSERT_TRUE(f.Close());
}

TEST_F(MemoryFileioTest, OpenForReadingNoSuchFile) {
  FileIO f;
  ASSERT_FALSE(f.Open("blah2", mako::FileIO::AccessMode::kRead));
  ASSERT_TRUE(f.Close());
}

TEST_F(MemoryFileioTest, MakeInstance) {
  FileIO f;
  std::unique_ptr<mako::FileIO> f2 = f.MakeInstance();
  ASSERT_TRUE(f2->Open("/tmp/blah", mako::FileIO::AccessMode::kWrite));
}

TEST_F(MemoryFileioTest, WriteWithoutOpen) {
  FileIO f;
  std::string file_path = "/tmp/blah";
  ASSERT_TRUE(f.Open(file_path, mako::FileIO::AccessMode::kWrite));
  KeyedValue k;
  k.set_value_key("1");
  k.set_value(1);
  ASSERT_TRUE(f.Write(k));
  ASSERT_TRUE(f.Close());

  ASSERT_FALSE(f.Read(&k));
}

TEST_F(MemoryFileioTest, ReadWithoutOpen) {
  FileIO f;
  KeyedValue k;
  k.set_value_key("k");
  k.set_value(1);
  ASSERT_FALSE(f.Write(k));
  f.Close();
}

TEST_F(MemoryFileioTest, DeleteNoSuchFile) {
  FileIO f;
  ASSERT_FALSE(f.Delete("/tmp/nosuch/file"));
}

TEST_F(MemoryFileioTest, CannotReadFromDeletedFile) {
  FileIO f;
  std::string file_path = "/tmp/blah";
  ASSERT_TRUE(f.Open(file_path, mako::FileIO::AccessMode::kWrite));
  KeyedValue k;
  k.set_value_key("1");
  k.set_value(1);
  ASSERT_TRUE(f.Write(k));
  ASSERT_TRUE(f.Close());

  ASSERT_TRUE(f.Delete(file_path));

  // Cannot read now.
  ASSERT_FALSE(f.Open(file_path, mako::FileIO::AccessMode::kRead));

  // But can write
  ASSERT_TRUE(f.Open(file_path, mako::FileIO::AccessMode::kWrite));
}

TEST_F(MemoryFileioTest, DeleteFileWhileReading) {
  std::string file_path = "/tmp/blah";

  FileIO f;
  ASSERT_TRUE(f.Open(file_path, mako::FileIO::AccessMode::kWrite));
  KeyedValue k;
  k.set_value_key("1");
  k.set_value(1);
  ASSERT_TRUE(f.Write(k));
  k.set_value_key("2");
  k.set_value(2);
  ASSERT_TRUE(f.Close());

  // Reading handle
  ASSERT_TRUE(f.Open(file_path, mako::FileIO::AccessMode::kRead));
  KeyedValue k2;
  ASSERT_TRUE(f.Read(&k2));

  // Current file handle won't delete while file is open
  ASSERT_FALSE(f.Delete(file_path));

  // Delete file while handle is still open.
  FileIO f2;
  ASSERT_TRUE(f2.Delete(file_path));

  // Next read fails
  ASSERT_FALSE(f.Read(&k));
  f.Close();

  // File is gone, can't open for reading.
  ASSERT_FALSE(f.Open(file_path, mako::FileIO::AccessMode::kRead));

  // But can write
  ASSERT_TRUE(f.Open(file_path, mako::FileIO::AccessMode::kWrite));
}

TEST_F(MemoryFileioTest, DeleteFileWhileWriting) {
  std::string file_path = "/tmp/blah";

  // Handle that is writing to file
  FileIO f;
  ASSERT_TRUE(f.Open(file_path, mako::FileIO::AccessMode::kWrite));
  KeyedValue k;
  k.set_value_key("1");
  k.set_value(1);
  ASSERT_TRUE(f.Write(k));

  // Current file handle won't delete while file is open
  ASSERT_FALSE(f.Delete(file_path));

  // Delete file while handle is still open.
  FileIO f2;
  ASSERT_TRUE(f2.Delete(file_path));

  // Next write fails
  ASSERT_FALSE(f.Write(k));
  f.Close();

  // File is gone, can't open for reading.
  ASSERT_FALSE(f.Open(file_path, mako::FileIO::AccessMode::kRead));

  // But can write
  ASSERT_TRUE(f.Open(file_path, mako::FileIO::AccessMode::kWrite));
}

TEST_F(MemoryFileioTest, WriteRecordsReadThemBack) {
  FileIO f;
  ASSERT_TRUE(f.Open("/tmp/blah", mako::FileIO::AccessMode::kWrite));
  int num_writes = 3;

  // Write some records
  for (int i = 0; i < num_writes; i++) {
    KeyedValue k;
    k.set_value_key(std::to_string(i));
    k.set_value(i);
    f.Write(k);
  }
  ASSERT_TRUE(f.Close());

  // Read them back
  ASSERT_TRUE(f.Open("/tmp/blah", mako::FileIO::AccessMode::kRead));
  KeyedValue k;
  for (int i = 0; i < num_writes; i++) {
    ASSERT_TRUE(f.Read(&k));
    ASSERT_EQ(k.value(), i);
    ASSERT_EQ(k.value_key(), std::to_string(i));
  }

  // no more records left.
  ASSERT_FALSE(f.Read(&k));
  ASSERT_TRUE(f.ReadEOF());

  ASSERT_TRUE(f.Close());
  ASSERT_TRUE(f.Error().empty());
}

TEST_F(MemoryFileioTest, WriteOverwrites) {
  FileIO f;
  ASSERT_TRUE(f.Open("/tmp/blah", mako::FileIO::AccessMode::kWrite));

  // Write single record
  KeyedValue k;
  k.set_value_key("1");
  k.set_value(1);
  ASSERT_TRUE(f.Write(k));
  f.Close();

  // Write another record
  ASSERT_TRUE(f.Open("/tmp/blah", mako::FileIO::AccessMode::kWrite));
  k.set_value_key("2");
  k.set_value(2);
  ASSERT_TRUE(f.Write(k));
  f.Close();

  // Read back second write value
  FileIO f2;
  ASSERT_TRUE(f2.Open("/tmp/blah", mako::FileIO::AccessMode::kRead));
  KeyedValue k2;
  ASSERT_TRUE(f2.Read(&k2));
  ASSERT_EQ(k2.value(), 2);
  ASSERT_EQ(k2.value_key(), "2");

  // No records left.
  ASSERT_FALSE(f2.Read(&k2));
  ASSERT_TRUE(f2.ReadEOF());

  f2.Close();
}

TEST_F(MemoryFileioTest, AppendOpensFile) {
  FileIO f;
  ASSERT_TRUE(f.Open("/tmp/blah", mako::FileIO::AccessMode::kAppend));

  // Write single record
  KeyedValue k;
  k.set_value_key("1");
  k.set_value(1);
  ASSERT_TRUE(f.Write(k));
  f.Close();

  // Read back written value.
  FileIO f2;
  ASSERT_TRUE(f2.Open("/tmp/blah", mako::FileIO::AccessMode::kRead));
  KeyedValue k2;
  ASSERT_TRUE(f2.Read(&k2));
  ASSERT_EQ(k2.value(), 1);
  ASSERT_EQ(k2.value_key(), "1");
}

TEST_F(MemoryFileioTest, AppendDoesntOverwrite) {
  FileIO f;
  ASSERT_TRUE(f.Open("/tmp/blah", mako::FileIO::AccessMode::kWrite));

  // Write single record
  KeyedValue k;
  k.set_value_key("1");
  k.set_value(1);
  ASSERT_TRUE(f.Write(k));
  f.Close();

  // Write another record
  ASSERT_TRUE(f.Open("/tmp/blah", mako::FileIO::AccessMode::kAppend));
  k.set_value_key("2");
  k.set_value(2);
  ASSERT_TRUE(f.Write(k));
  f.Close();

  // Read back second first write value, then second.
  FileIO f2;
  ASSERT_TRUE(f2.Open("/tmp/blah", mako::FileIO::AccessMode::kRead));
  KeyedValue k2;
  ASSERT_TRUE(f2.Read(&k2));
  ASSERT_EQ(k2.value(), 1);
  ASSERT_EQ(k2.value_key(), "1");
  k2.Clear();
  ASSERT_TRUE(f2.Read(&k2));
  ASSERT_EQ(k2.value(), 2);
  ASSERT_EQ(k2.value_key(), "2");

  // No records left.
  ASSERT_FALSE(f2.Read(&k2));
  ASSERT_TRUE(f2.ReadEOF());

  f2.Close();
}

TEST_F(MemoryFileioTest, InjectedErrorsTest) {
  FileIO f;
  f.set_open_error("open");
  f.set_read_error("read");
  f.set_write_error("write");
  f.set_delete_error("delete");
  f.set_close_error("close");

  ASSERT_FALSE(f.Open("/tmp/blah", mako::FileIO::AccessMode::kRead));
  ASSERT_EQ(f.Error(), "memory_fileio::FileIO open");

  KeyedValue k;
  ASSERT_FALSE(f.Read(&k));
  ASSERT_EQ(f.Error(), "memory_fileio::FileIO read");
  ASSERT_FALSE(f.Write(k));
  ASSERT_EQ(f.Error(), "memory_fileio::FileIO write");

  std::string serialized_record;
  ASSERT_FALSE(f.Read(&serialized_record));
  ASSERT_EQ(f.Error(), "memory_fileio::FileIO read");
  ASSERT_FALSE(f.Write(serialized_record));
  ASSERT_EQ(f.Error(), "memory_fileio::FileIO write");

  ASSERT_FALSE(f.Close());
  ASSERT_EQ(f.Error(), "memory_fileio::FileIO close");

  ASSERT_FALSE(f.Delete("/tmp/blah"));
  ASSERT_EQ(f.Error(), "memory_fileio::FileIO delete");
}

TEST_F(MemoryFileioTest, InjectedErrorsMakeInstanceTest) {
  FileIO f;
  f.set_open_error("open");

  auto copy = f.MakeInstance();
  ASSERT_FALSE(copy->Open("/tmp/blah", mako::FileIO::AccessMode::kRead));
  ASSERT_EQ(copy->Error(), "memory_fileio::FileIO open");
}

TEST(FileIOTest, WriteProtoReadString) {
  std::string path = "write-proto-read-string";
  mako::memory_fileio::FileIO fio;
  ASSERT_TRUE(fio.Open(path, mako::FileIO::AccessMode::kWrite))
      << fio.Error();

  mako::KeyedValue k;
  k.set_value(17);
  ASSERT_TRUE(fio.Write(k)) << fio.Error();
  ASSERT_TRUE(fio.Close()) << fio.Error();

  ASSERT_TRUE(fio.Open(path, mako::FileIO::AccessMode::kRead))
      << fio.Error();

  mako::KeyedValue k2;
  std::string s;
  ASSERT_TRUE(fio.Read(&s)) << fio.Error();

  k2.ParseFromString(s);
  ASSERT_EQ(17, k2.value());
  ASSERT_TRUE(fio.Close()) << fio.Error();
}

TEST(FileIOTest, WriteStringReadProto) {
  std::string path = "write-std::string-read-proto";
  mako::memory_fileio::FileIO fio;
  ASSERT_TRUE(fio.Open(path, mako::FileIO::AccessMode::kWrite))
      << fio.Error();

  mako::KeyedValue k;
  k.set_value(17);
  ASSERT_TRUE(fio.Write(k.SerializeAsString())) << fio.Error();
  ASSERT_TRUE(fio.Close()) << fio.Error();

  ASSERT_TRUE(fio.Open(path, mako::FileIO::AccessMode::kRead))
      << fio.Error();

  mako::KeyedValue k2;
  ASSERT_TRUE(fio.Read(&k2)) << fio.Error();

  ASSERT_EQ(17, k2.value());
  ASSERT_TRUE(fio.Close()) << fio.Error();
}
}  // namespace memory_fileio
}  // namespace mako
