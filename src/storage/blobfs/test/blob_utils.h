// Copyright 2019 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

#ifndef SRC_STORAGE_BLOBFS_TEST_BLOB_UTILS_H_
#define SRC_STORAGE_BLOBFS_TEST_BLOB_UTILS_H_

#include <lib/fdio/io.h>

#include <cstring>
#include <memory>
#include <string>

#include <digest/digest.h>
#include <fbl/unique_fd.h>

#include "src/storage/blobfs/blob-layout.h"

namespace blobfs {

using BlobSrcFunction = std::function<void(char* data, size_t length)>;

// An in-memory representation of a blob.
struct BlobInfo {
  char path[PATH_MAX];
  std::unique_ptr<char[]> merkle;
  size_t size_merkle;
  std::unique_ptr<char[]> data;
  size_t size_data;
};

template <typename T, typename U>
int StreamAll(T func, int fd, U* buf, size_t max) {
  size_t n = 0;
  while (n != max) {
    ssize_t d = func(fd, &buf[n], max - n);
    if (d < 0) {
      return -1;
    }
    n += d;
  }
  return 0;
}

// Fills the provided buffer with a single character |C|.
template <char C>
void CharFill(char* data, size_t length) {
  memset(data, C, length);
}

// Fills the provided buffer with random data. The contents of the buffer are
// repeatable for any iteration of the test, provided that srand() is seeded with
// the test framework's seed.
void RandomFill(char* data, size_t length);

// Generates a blob with data generated by a user-provided function.
// TODO(fxbug.dev/63295): Return zx::status<std::unique_ptr<BlobInfo>> and remove the
// |blob_layout_format| param.
void GenerateBlob(BlobSrcFunction data_generator, const std::string& mount_path, size_t data_size,
                  BlobLayoutFormat blob_layout_format, std::unique_ptr<BlobInfo>* out);

// Generates a blob with data generated by a user-provided function.
void GenerateBlob(BlobSrcFunction data_generator, const std::string& mount_path, size_t data_size,
                  std::unique_ptr<BlobInfo>* out);

// Generates a blob with random data.
// TODO(fxbug.dev/63295): Return zx::status<std::unique_ptr<BlobInfo>> and remove the
// |blob_layout_format| param.
void GenerateRandomBlob(const std::string& mount_path, size_t data_size,
                        BlobLayoutFormat blob_layout_format, std::unique_ptr<BlobInfo>* out);

// Generates a blob with random data.
void GenerateRandomBlob(const std::string& mount_path, size_t data_size,
                        std::unique_ptr<BlobInfo>* out);

// Generates a blob with realistic data (derived from, for example, an ELF binary).
// This is suitable for test cases which aim to exercise compression, since the blob will compress
// a realistic amount.
// TODO(fxbug.dev/63295): Return zx::status<std::unique_ptr<BlobInfo>> and remove the
// |blob_layout_format| param.
void GenerateRealisticBlob(const std::string& mount_path, size_t data_size,
                           BlobLayoutFormat blob_layout_format, std::unique_ptr<BlobInfo>* out);

// Verifies that a file contains the data in the provided buffer.
void VerifyContents(int fd, const char* data, size_t data_size);

// Creates an open blob with the provided Merkle tree + Data, and reads back to verify the data.
// Asserts (via ASSERT_* in gtest) that the write and read succeeded.
// TODO(jfsulliv): Return a status, or change the name to indicate that this will assert on failure.
void MakeBlob(const BlobInfo* info, fbl::unique_fd* fd);

// Returns the name of |format| for use in parameterized tests.
std::string GetBlobLayoutFormatNameForTests(BlobLayoutFormat format);

// An in-memory representation of a Merkle tree.
struct MerkleTreeInfo {
  std::unique_ptr<uint8_t[]> merkle_tree;
  uint64_t merkle_tree_size;
  Digest root;
};

// Constructs a Merkle tree for |data|.
zx::status<MerkleTreeInfo> CreateMerkleTree(const uint8_t* data, int64_t data_size,
                                            bool use_compact_format);

}  // namespace blobfs

#endif  // SRC_STORAGE_BLOBFS_TEST_BLOB_UTILS_H_
