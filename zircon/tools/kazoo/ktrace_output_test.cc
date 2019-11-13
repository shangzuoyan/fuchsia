// Copyright 2019 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

#include "tools/kazoo/outputs.h"
#include "tools/kazoo/syscall_library.h"
#include "tools/kazoo/test.h"
#include "tools/kazoo/test_ir_test_one_protocol_one_method.test.h"

namespace {

TEST(KtraceOutput, Simple) {
  SyscallLibrary library;
  ASSERT_TRUE(SyscallLibraryLoader::FromJson(k_test_one_protocol_one_method, &library));

  StringWriter writer;
  ASSERT_TRUE(KtraceOutput(library, &writer));

  EXPECT_EQ(writer.Out(),
            R"(// Copyright 2019 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// WARNING: THIS FILE IS MACHINE GENERATED BY //tools/kazoo. DO NOT EDIT.

{0, 1, "single_do_thing"},
)");
}

}  // namespace
