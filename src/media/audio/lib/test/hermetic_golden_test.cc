// Copyright 2020 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

#include "src/media/audio/lib/test/hermetic_golden_test.h"

#include <fuchsia/media/cpp/fidl.h>
#include <lib/syslog/cpp/macros.h>

#include "src/lib/files/file.h"
#include "src/lib/files/path.h"
#include "src/lib/fxl/strings/string_printf.h"
#include "src/media/audio/lib/analysis/analysis.h"
#include "src/media/audio/lib/analysis/generators.h"
#include "src/media/audio/lib/format/audio_buffer.h"
#include "src/media/audio/lib/test/comparators.h"
#include "src/media/audio/lib/test/hermetic_pipeline_test.h"
#include "src/media/audio/lib/test/renderer_shim.h"
#include "src/media/audio/lib/wav/wav_writer.h"

using ASF = fuchsia::media::AudioSampleFormat;

namespace media::audio::test {

namespace {

// Using a macro here to preserve line numbers in the test error messages.
#define EXPECT_WITHIN_RELATIVE_ERROR(actual, expected, threshold)                          \
  do {                                                                                     \
    auto label =                                                                           \
        fxl::StringPrintf("\n  %s = %f\n  %s = %f", #actual, actual, #expected, expected); \
    EXPECT_NE(expected, 0) << label;                                                       \
    if (expected != 0) {                                                                   \
      auto err = abs(actual - expected) / expected;                                        \
      EXPECT_LE(err, threshold) << label;                                                  \
    }                                                                                      \
  } while (0)

// WaveformTestRunner wraps some methods used by RunTestCase.
template <ASF InputFormat, ASF OutputFormat>
class WaveformTestRunner {
 public:
  WaveformTestRunner(const HermeticGoldenTest::WaveformTestCase<InputFormat, OutputFormat>& tc)
      : tc_(tc) {}

  void CompareRMSE(AudioBufferSlice<OutputFormat> actual, AudioBufferSlice<OutputFormat> expected);
  void CompareRMS(AudioBufferSlice<OutputFormat> actual, AudioBufferSlice<OutputFormat> expected);
  void CompareFreqs(AudioBufferSlice<OutputFormat> actual, AudioBufferSlice<OutputFormat> expected,
                    std::vector<size_t> hz_signals);

  void ExpectSilence(AudioBufferSlice<OutputFormat> actual);

 private:
  const HermeticGoldenTest::WaveformTestCase<InputFormat, OutputFormat>& tc_;
};

template <ASF InputFormat, ASF OutputFormat>
void WaveformTestRunner<InputFormat, OutputFormat>::CompareRMSE(
    AudioBufferSlice<OutputFormat> actual, AudioBufferSlice<OutputFormat> expected) {
  CompareAudioBuffers(actual, expected,
                      {
                          .max_relative_error = tc_.max_relative_rms_error,
                          .test_label = "check data",
                          .num_frames_per_packet = expected.format().frames_per_second() / 1000 *
                                                   RendererShimImpl::kPacketMs,
                      });
}

template <ASF InputFormat, ASF OutputFormat>
void WaveformTestRunner<InputFormat, OutputFormat>::CompareRMS(
    AudioBufferSlice<OutputFormat> actual, AudioBufferSlice<OutputFormat> expected) {
  auto expected_rms = MeasureAudioRMS(expected);
  auto actual_rms = MeasureAudioRMS(actual);
  EXPECT_WITHIN_RELATIVE_ERROR(actual_rms, expected_rms, tc_.max_relative_rms);
}

template <ASF InputFormat, ASF OutputFormat>
void WaveformTestRunner<InputFormat, OutputFormat>::CompareFreqs(
    AudioBufferSlice<OutputFormat> actual, AudioBufferSlice<OutputFormat> expected,
    std::vector<size_t> hz_signals) {
  ASSERT_EQ(expected.NumFrames(), actual.NumFrames());

  // FFT requires a power-of-2 number of samples.
  auto expected_buf = PadToNearestPower2(expected);
  auto actual_buf = PadToNearestPower2(actual);
  expected = AudioBufferSlice(&expected_buf);
  actual = AudioBufferSlice(&actual_buf);

  // Translate hz to periods.
  std::unordered_set<size_t> freqs_in_unit_periods;
  for (auto hz : hz_signals) {
    ASSERT_GT(hz, 0u);

    // Frames per period at frequency hz.
    size_t fpp = expected.format().frames_per_second() / hz;

    // If there are an integer number of periods, we can precisely measure the magnitude at hz.
    // Otherwise, the magnitude will be smeared between the two adjacent integers.
    size_t periods = expected.NumFrames() / fpp;
    freqs_in_unit_periods.insert(periods);
    if (expected.NumFrames() % fpp > 0) {
      freqs_in_unit_periods.insert(periods + 1);
    }
  }

  auto expected_result = MeasureAudioFreqs(expected, freqs_in_unit_periods);
  auto actual_result = MeasureAudioFreqs(actual, freqs_in_unit_periods);

  EXPECT_WITHIN_RELATIVE_ERROR(actual_result.total_magn_signal, expected_result.total_magn_signal,
                               tc_.max_relative_signal_error);
  EXPECT_WITHIN_RELATIVE_ERROR(actual_result.total_magn_other, expected_result.total_magn_other,
                               tc_.max_relative_other_error);

  for (auto periods : freqs_in_unit_periods) {
    auto hz = static_cast<double>(periods) *
              static_cast<double>(expected.format().frames_per_second()) /
              static_cast<double>(expected.NumFrames());
    SCOPED_TRACE(testing::Message() << "Frequency " << periods << " periods, " << hz << " hz");

    EXPECT_WITHIN_RELATIVE_ERROR(actual_result.magnitudes[periods],
                                 expected_result.magnitudes[periods],
                                 tc_.max_relative_signal_error);

    EXPECT_WITHIN_RELATIVE_ERROR(actual_result.phases[periods], expected_result.phases[periods],
                                 tc_.max_relative_signal_phase_error);
  }
}

template <ASF InputFormat, ASF OutputFormat>
void WaveformTestRunner<InputFormat, OutputFormat>::ExpectSilence(
    AudioBufferSlice<OutputFormat> actual) {
  CompareAudioBuffers(actual, AudioBufferSlice<OutputFormat>(),
                      {
                          .test_label = "check silence",
                          .num_frames_per_packet = actual.format().frames_per_second() / 1000 *
                                                   RendererShimImpl::kPacketMs,
                      });
}

}  // namespace

template <ASF InputFormat, ASF OutputFormat>
void HermeticGoldenTest::RunWaveformTest(
    const HermeticGoldenTest::WaveformTestCase<InputFormat, OutputFormat>& tc) {
  WaveformTestRunner<InputFormat, OutputFormat> runner(tc);

  const auto& input = tc.input;
  const auto& expected_output = tc.expected_output;

  auto device = CreateOutput(AUDIO_STREAM_UNIQUE_ID_BUILTIN_SPEAKERS, expected_output.format(),
                             AddSlackToOutputFrames(expected_output.NumFrames()), std::nullopt,
                             tc.pipeline.output_device_gain_db);
  auto renderer = CreateAudioRenderer(input.format(), input.NumFrames());

  // Render the input at a time such that the first frame of audio will be rendered into
  // the first frame of the ring buffer. We need to synchronize with the ring buffer, then
  // leave some silence to account for ring in.
  auto packets = renderer->AppendPackets({&input});
  auto min_start_time = zx::clock::get_monotonic() + renderer->min_lead_time() + zx::msec(20);
  auto start_time =
      device->NextSynchronizedTimestamp(min_start_time) +
      zx::nsec(renderer->format().frames_per_ns().Inverse().Scale(tc.pipeline.neg_filter_width));
  renderer->Play(this, start_time, 0);
  renderer->WaitForPackets(this, packets);

  // The ring buffer should contain the expected output followed by silence.
  auto ring_buffer = device->SnapshotRingBuffer();
  auto num_data_frames = expected_output.NumFrames();
  auto output_data = AudioBufferSlice(&ring_buffer, 0, num_data_frames);
  auto output_silence = AudioBufferSlice(&ring_buffer, num_data_frames, device->frame_count());

  if (save_input_and_output_files_) {
    WriteWavFile<InputFormat>(tc.test_name, "input", &input);
    WriteWavFile<OutputFormat>(tc.test_name, "ring_buffer", &ring_buffer);
    WriteWavFile<OutputFormat>(tc.test_name, "output", output_data);
    WriteWavFile<OutputFormat>(tc.test_name, "expected_output", &expected_output);
  }

  runner.CompareRMSE(&expected_output, output_data);
  runner.ExpectSilence(output_silence);

  for (size_t chan = 0; chan < expected_output.format().channels(); chan++) {
    auto expected_chan = AudioBufferSlice<OutputFormat>(&expected_output).GetChannel(chan);
    auto output_chan = output_data.GetChannel(chan);
    SCOPED_TRACE(testing::Message() << "Channel " << chan);
    runner.CompareRMS(&output_chan, &expected_chan);
    runner.CompareFreqs(&output_chan, &expected_chan, tc.frequencies_hz_to_analyze);
  }
}

// TODO: remove HermeticGoldenTest::RunImpulseTest, after clients move to HermeticImpulseTest.
template <ASF InputFormat, ASF OutputFormat>
void HermeticGoldenTest::RunImpulseTest(
    const HermeticGoldenTest::ImpulseTestCase<InputFormat, OutputFormat>& tc) {
  // Compute the number of input frames.
  auto start_of_last_impulse = tc.impulse_locations_in_frames.back();
  auto num_input_frames = start_of_last_impulse + tc.impulse_width_in_frames +
                          tc.pipeline.pos_filter_width + tc.pipeline.neg_filter_width;

  // Helper to translate from an input frame number to an output frame number.
  auto input_frame_to_output_frame = [&tc](size_t input_frame) {
    auto input_fps = static_cast<double>(tc.input_format.frames_per_second());
    auto output_fps = static_cast<double>(tc.output_format.frames_per_second());
    return static_cast<size_t>(std::ceil(output_fps / input_fps * input_frame));
  };

  auto num_output_frames = input_frame_to_output_frame(num_input_frames);
  auto device = CreateOutput(AUDIO_STREAM_UNIQUE_ID_BUILTIN_SPEAKERS, tc.output_format,
                             AddSlackToOutputFrames(num_output_frames), std::nullopt,
                             tc.pipeline.output_device_gain_db);
  auto renderer = CreateAudioRenderer(tc.input_format, num_input_frames);

  // Write all of the impulses to an input buffer so we can easily write the full
  // input to a WAV file for debugging. Include silence at the beginning to account
  // for ring in; this allows us to align the input and output WAV files.
  auto input_impulse_start = tc.pipeline.neg_filter_width;
  AudioBuffer<InputFormat> input(tc.input_format, num_input_frames);
  for (auto start_frame : tc.impulse_locations_in_frames) {
    start_frame += input_impulse_start;
    for (size_t f = start_frame; f < start_frame + tc.impulse_width_in_frames; f++) {
      for (size_t c = 0; c < tc.input_format.channels(); c++) {
        input.samples()[input.SampleIndex(f, c)] = tc.impulse_magnitude;
      }
    }
  }

  // Render the input at a time such that the first frame of audio will be rendered into
  // the first frame of the ring buffer.
  auto packets = renderer->AppendPackets({&input});
  renderer->PlaySynchronized(this, device, 0);
  renderer->WaitForPackets(this, packets);

  auto ring_buffer = device->SnapshotRingBuffer();

  // The ring buffer should contain the expected sequence of impulses.
  // Due to smoothing effects, the detected leading edge of each impulse might be offset
  // slightly from the expected location, however each impulse should be offset by the
  // same amount. Empirically, we see offsets as high as 0.5ms. Allow up to 1ms.
  ssize_t max_impulse_offset_frames = tc.output_format.frames_per_ns().Scale(zx::msec(1).get());
  std::unordered_map<size_t, ssize_t> first_impulse_offset_per_channel;
  size_t search_start_frame = 0;
  size_t search_end_frame = 0;

  for (size_t k = 0; k < tc.impulse_locations_in_frames.size(); k++) {
    // End this search halfway between impulses k and k+1.
    size_t input_next_midpoint_frame;
    if (k + 1 < tc.impulse_locations_in_frames.size()) {
      auto curr = input_impulse_start + tc.impulse_locations_in_frames[k];
      auto next = input_impulse_start + tc.impulse_locations_in_frames[k + 1];
      input_next_midpoint_frame = curr + (next - curr) / 2;
    } else {
      input_next_midpoint_frame = num_input_frames;
    }
    search_start_frame = search_end_frame;
    search_end_frame = input_frame_to_output_frame(input_next_midpoint_frame);

    // We expect zero noise in the output.
    constexpr auto kNoiseFloor = 0;

    // Impulse should be at this frame +/- max_impulse_offset_frames.
    auto expected_output_frame =
        input_frame_to_output_frame(input_impulse_start + tc.impulse_locations_in_frames[k]);

    // Test each channel.
    for (size_t chan = 0; chan < tc.output_format.channels(); chan++) {
      SCOPED_TRACE(testing::Message() << "Channel " << chan);
      auto output_chan = AudioBufferSlice<OutputFormat>(&ring_buffer).GetChannel(chan);
      auto slice = AudioBufferSlice(&output_chan, search_start_frame, search_end_frame);
      auto relative_output_frame = FindImpulseLeadingEdge(slice, kNoiseFloor);
      if (!relative_output_frame) {
        ADD_FAILURE() << "Could not find impulse " << k << " in ring buffer\n"
                      << "Expected at ring buffer frame " << expected_output_frame << "\n"
                      << "Ring buffer is:";
        output_chan.Display(search_start_frame, search_end_frame);
        continue;
      }
      auto output_frame = *relative_output_frame + search_start_frame;
      if (k == 0) {
        // First impulse decides the offset.
        auto offset =
            static_cast<ssize_t>(output_frame) - static_cast<ssize_t>(expected_output_frame);
        EXPECT_LE(std::abs(offset), max_impulse_offset_frames)
            << "Found impulse " << k << " at an unexpected location: at frame " << output_frame
            << ", expected within " << max_impulse_offset_frames << " frames of "
            << expected_output_frame;
        first_impulse_offset_per_channel[chan] = offset;
      } else {
        // Other impulses should have the same offset.
        auto expected_offset = first_impulse_offset_per_channel[chan];
        EXPECT_EQ(expected_output_frame + expected_offset, output_frame)
            << "Found impulse " << k << " at an unexpected location; expected_offset is "
            << expected_offset;
      }
    }
  }

  if (save_input_and_output_files_) {
    WriteWavFile<InputFormat>(tc.test_name, "input", &input);
    WriteWavFile<OutputFormat>(tc.test_name, "ring_buffer", &ring_buffer);
  }
}

// Explicitly instantiate (almost) all possible implementations.
// We intentionally don't instantiate implementations with OutputFormat = UNSIGNED_8
// because such hardware is no longer in use, therefore it's not worth testing.
//
// TODO: remove HermeticGoldenTest::RunImpulseTest below, after clients move to HermeticImpulseTest.
#define INSTANTIATE(InputFormat, OutputFormat)                                  \
  template void HermeticGoldenTest::RunWaveformTest<InputFormat, OutputFormat>( \
      const WaveformTestCase<InputFormat, OutputFormat>& tc);                   \
  template void HermeticGoldenTest::RunImpulseTest<InputFormat, OutputFormat>(  \
      const ImpulseTestCase<InputFormat, OutputFormat>& tc);

INSTANTIATE(ASF::UNSIGNED_8, ASF::SIGNED_16)
INSTANTIATE(ASF::UNSIGNED_8, ASF::SIGNED_24_IN_32)
INSTANTIATE(ASF::UNSIGNED_8, ASF::FLOAT)

INSTANTIATE(ASF::SIGNED_16, ASF::SIGNED_16)
INSTANTIATE(ASF::SIGNED_16, ASF::SIGNED_24_IN_32)
INSTANTIATE(ASF::SIGNED_16, ASF::FLOAT)

INSTANTIATE(ASF::SIGNED_24_IN_32, ASF::SIGNED_16)
INSTANTIATE(ASF::SIGNED_24_IN_32, ASF::SIGNED_24_IN_32)
INSTANTIATE(ASF::SIGNED_24_IN_32, ASF::FLOAT)

INSTANTIATE(ASF::FLOAT, ASF::SIGNED_16)
INSTANTIATE(ASF::FLOAT, ASF::SIGNED_24_IN_32)
INSTANTIATE(ASF::FLOAT, ASF::FLOAT)

}  // namespace media::audio::test
