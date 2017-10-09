// Copyright 2016 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

#include "peridot/bin/suggestion_engine/ask_subscriber.h"

namespace maxwell {

AskSubscriber::AskSubscriber(
    const RankedSuggestions* ranked_suggestions,
    AskDispatcher* ask_dispatcher,
    fidl::InterfaceRequest<TranscriptionListener> transcription_listener,
    fidl::InterfaceHandle<SuggestionListener> listener,
    fidl::InterfaceRequest<AskController> controller)
    : BoundWindowedSuggestionSubscriber(ranked_suggestions,
                                        std::move(listener),
                                        std::move(controller)),
      ask_dispatcher_(ask_dispatcher) {}

void AskSubscriber::SetUserInput(UserInputPtr input) {
  ask_dispatcher_->DispatchAsk(std::move(input));
}

void AskSubscriber::BeginSpeechCapture(
    fidl::InterfaceHandle<TranscriptionListener> transcription_listener) {
  ask_dispatcher_->BeginSpeechCapture(std::move(transcription_listener));
}

}  // namespace maxwell
