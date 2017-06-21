// Copyright 2017 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

#include "apps/ledger/src/app/integration_tests/integration_test.h"
#include "apps/ledger/src/app/integration_tests/test_utils.h"
#include "apps/ledger/src/convert/convert.h"
#include "apps/modular/services/auth/token_provider.fidl.h"

namespace ledger {
namespace integration_tests {
namespace {

class LedgerRepositoryIntegrationTest : public IntegrationTest {
 public:
  LedgerRepositoryIntegrationTest() {}
  ~LedgerRepositoryIntegrationTest() override {}

 private:
  FTL_DISALLOW_COPY_AND_ASSIGN(LedgerRepositoryIntegrationTest);
};

// Verifies that the LedgerRepository and its children are shut down on token
// manager connection error.
TEST_F(LedgerRepositoryIntegrationTest, ShutDownOnTokenProviderError) {
  const auto timeout = ftl::TimeDelta::FromSeconds(1);
  files::ScopedTempDir tmp_dir;
  Status status;
  modular::auth::TokenProviderPtr token_provider_ptr;
  auto token_provider_request = token_provider_ptr.NewRequest();

  FirebaseConfigPtr firebase_config = FirebaseConfig::New();
  firebase_config->server_id = "server-id";
  firebase_config->api_key = "";

  LedgerRepositoryPtr repository;
  ledger_repository_factory_->GetRepository(
      tmp_dir.path(), std::move(firebase_config), std::move(token_provider_ptr),
      repository.NewRequest(), [&status](Status s) { status = s; });
  ASSERT_TRUE(
      ledger_repository_factory_.WaitForIncomingResponseWithTimeout(timeout));
  EXPECT_EQ(Status::OK, status);
  bool repository_disconnected = false;
  repository.set_connection_error_handler(
      [&repository_disconnected] { repository_disconnected = true; });

  LedgerPtr ledger;
  repository->GetLedger(convert::ToArray("bla"), ledger.NewRequest(),
                        [&status](Status s) { status = s; });
  ASSERT_TRUE(repository.WaitForIncomingResponseWithTimeout(timeout));
  EXPECT_EQ(Status::OK, status);
  bool ledger_disconnected = false;
  ledger.set_connection_error_handler(
      [&ledger_disconnected] { ledger_disconnected = true; });

  token_provider_request.PassChannel().reset();

  ASSERT_FALSE(ledger.WaitForIncomingResponseWithTimeout(timeout));
  EXPECT_TRUE(ledger_disconnected);

  ASSERT_FALSE(repository.WaitForIncomingResponseWithTimeout(timeout));
  EXPECT_TRUE(repository_disconnected);
}

}  // namespace
}  // namespace integration_tests
}  // namespace ledger
