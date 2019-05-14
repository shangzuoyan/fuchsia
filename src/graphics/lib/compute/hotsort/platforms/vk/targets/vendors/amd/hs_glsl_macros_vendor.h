// Copyright 2019 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

#ifndef SRC_GRAPHICS_LIB_COMPUTE_HOTSORT_PLATFORMS_VK_TARGETS_VENDORS_AMD_HS_GLSL_MACROS_VENDOR_H_
#define SRC_GRAPHICS_LIB_COMPUTE_HOTSORT_PLATFORMS_VK_TARGETS_VENDORS_AMD_HS_GLSL_MACROS_VENDOR_H_

//
//
//

#include "hs_glsl_macros.h"

//
// OVERRIDE SUBGROUP LANE ID
//

#define HS_GLSL_SUBGROUP_SIZE()
#define HS_SUBGROUP_PREAMBLE()

#define HS_SUBGROUP_ID() gl_SubgroupID
#define HS_SUBGROUP_LANE_ID() gl_SubgroupInvocationID

//
// CHOOSE A COMPARE-EXCHANGE IMPLEMENTATION
//

#if (HS_KEY_DWORDS == 1)
#define HS_CMP_XCHG(a, b) HS_CMP_XCHG_V0(a, b)
#elif (HS_KEY_DWORDS == 2)
#define HS_CMP_XCHG(a, b) HS_CMP_XCHG_V3(a, b)
#endif

//
// CHOOSE A CONDITIONAL MIN/MAX IMPLEMENTATION
//

#if (HS_KEY_DWORDS == 1)
#define HS_COND_MIN_MAX(lt, a, b) HS_COND_MIN_MAX_V0(lt, a, b)
#elif (HS_KEY_DWORDS == 2)
#define HS_COND_MIN_MAX(lt, a, b) HS_COND_MIN_MAX_V0(lt, a, b)
#endif

//
//
//

#endif  // SRC_GRAPHICS_LIB_COMPUTE_HOTSORT_PLATFORMS_VK_TARGETS_VENDORS_AMD_HS_GLSL_MACROS_VENDOR_H_
