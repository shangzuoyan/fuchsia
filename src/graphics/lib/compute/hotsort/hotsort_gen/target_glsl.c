// Copyright 2019 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

#include <ctype.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

//
//
//

#include "common/macros.h"
#include "common/util.h"
#include "gen.h"
#include "transpose.h"

//
//
//

struct hsg_transpose_state
{
  FILE *                    header;
  struct hsg_config const * config;
};

static char
hsg_transpose_reg_prefix(uint32_t const cols_log2)
{
  return 'a' + (('r' + cols_log2 - 'a') % 26);
}

static void
hsg_transpose_blend(uint32_t const cols_log2,
                    uint32_t const row_ll,  // lower-left
                    uint32_t const row_ur,  // upper-right
                    void *         blend)
{
  struct hsg_transpose_state * const state = blend;

  // we're starting register names at '1' for now
  fprintf(state->header,
          "  HS_TRANSPOSE_BLEND( %c, %c, %2u, %3u, %3u ) \\\n",
          hsg_transpose_reg_prefix(cols_log2 - 1),
          hsg_transpose_reg_prefix(cols_log2),
          cols_log2,
          row_ll + 1,
          row_ur + 1);
}

static void
hsg_transpose_remap(uint32_t const row_from, uint32_t const row_to, void * remap)
{
  struct hsg_transpose_state * const state = remap;

  // we're starting register names at '1' for now
  fprintf(state->header,
          "  HS_TRANSPOSE_REMAP( %c, %3u, %3u )        \\\n",
          hsg_transpose_reg_prefix(state->config->warp.lanes_log2),
          row_from + 1,
          row_to + 1);
}

//
//
//

static void
hsg_copyright(FILE * file)
{
  fprintf(file,
          "// Copyright 2019 The Fuchsia Authors. All rights reserved.\n"
          "// Use of this source code is governed by a BSD-style license that can be\n"
          "// found in the LICENSE file.\n"
          "\n");
}

static void
hsg_macros(struct hsg_config const * const config, FILE * file)
{
  fprintf(file,
          "// preamble                                     \n"
          "#version 460                                    \n"
          "#extension GL_GOOGLE_include_directive : require\n"
          "                                                \n"
          "// target-specific config                       \n"
          "#include \"hs_config.h\"                        \n"
          "                                                \n"
          "// vendor<arch<target-specific macros           \n"
          "#include \"hs_glsl_macros_vendor.h\"            \n"
          "                                                \n"
          "//                                              \n"
          "//                                              \n"
          "//                                              \n"
          "\n");
}

//
//
//

static void
hsg_target_header_and_module(struct hsg_config const * const config)
{
  //
  // define header
  //
  {
    FILE * file = fopen("hs_target.h", "wb");

    hsg_copyright(file);

    fprintf(file,
            "#pragma once                                              \n"
            "                                                          \n"
            "//                                                        \n"
            "//                                                        \n"
            "//                                                        \n"
            "                                                          \n"
            "#include \"hotsort_vk_target.h\"                          \n"
            "                                                          \n"
            "//                                                        \n"
            "//                                                        \n"
            "//                                                        \n"
            "                                                          \n"
            "#undef  HS_TARGET_NAME                                    \n"
            "#define HS_TARGET_NAME %s                                 \n"
            "                                                          \n"
            "//                                                        \n"
            "//                                                        \n"
            "//                                                        \n"
            "                                                          \n"
            "#ifdef __cplusplus                                        \n"
            "extern \"C\" {                                            \n"
            "#endif                                                    \n"
            "                                                          \n"
            "//                                                        \n"
            "//                                                        \n"
            "//                                                        \n"
            "                                                          \n"
            "extern struct hotsort_vk_target const HS_TARGET_NAME;     \n"
            "                                                          \n"
            "//                                                        \n"
            "//                                                        \n"
            "//                                                        \n"
            "                                                          \n"
            "#ifdef __cplusplus                                        \n"
            "}                                                         \n"
            "#endif                                                    \n"
            "                                                          \n"
            "//                                                        \n"
            "//                                                        \n"
            "//                                                        \n",
            config->define.lower);

    fclose(file);
  }

  //
  // define module
  //
  {
    size_t const module_len  = strlen(config->define.lower) + strlen(".c") + 1;
    char * const module_name = malloc(module_len);

    sprintf(module_name, "%s.c", config->define.lower);

    FILE * file = fopen(module_name, "wb");

    hsg_copyright(file);

    fprintf(file,
            "//                                                   \n"
            "//                                                   \n"
            "//                                                   \n"
            "                                                     \n"
            "#include \"hs_target.h\"                             \n"
            "#include \"hs_config.h\"                             \n"
            "                                                     \n"
            "//                                                   \n"
            "//                                                   \n"
            "//                                                   \n"
            "                                                     \n"
            "struct hotsort_vk_target const HS_TARGET_NAME =      \n"
            "{                                                    \n"
            "  .config = {                                        \n"
            "#include \"hs_target_config_init.inl\"               \n"
            "  },                                                 \n"
            "                                                     \n"
            "  .modules = {                                       \n"
            "#include \"hs_modules.inl\"                          \n"
            "#ifdef HS_DUMP                                       \n"
            "    0                                                \n"
            "#endif                                               \n"
            "  }                                                  \n"
            "};                                                   \n"
            "                                                     \n"
            "#include \"hs_target_modules_dump.inl\"              \n"
            "                                                     \n"
            "//                                                   \n"
            "//                                                   \n"
            "//                                                   \n");

    fclose(file);

    free(module_name);
  }
}

//
//
//

struct hsg_target_state
{
  FILE * header;
  FILE * modules;
  FILE * source;
};

//
//
//

void
hsg_target_glsl(struct hsg_target * const       target,
                struct hsg_config const * const config,
                struct hsg_merge const * const  merge,
                struct hsg_op const * const     ops,
                uint32_t const                  depth)
{
  switch (ops->type)
    {
      case HSG_OP_TYPE_END:
        fprintf(target->state->source, "}\n");

        if (depth == 0)
          {
            fclose(target->state->source);
            target->state->source = NULL;
          }
        break;

      case HSG_OP_TYPE_BEGIN:
        fprintf(target->state->source, "{\n");
        break;

      case HSG_OP_TYPE_ELSE:
        fprintf(target->state->source, "else\n");
        break;

        case HSG_OP_TYPE_TARGET_BEGIN: {
          // create header and module
          hsg_target_header_and_module(config);

          // allocate state
          target->state = malloc(sizeof(*target->state));

          // allocate files
          target->state->header  = fopen("hs_config.h", "wb");
          target->state->modules = fopen("hs_modules.txt", "wb");

          hsg_copyright(target->state->header);

          // initialize header
          uint32_t const bc_max = msb_idx_u32(pow2_rd_u32(merge->warps));

          fprintf(target->state->header,
                  "#ifndef %s_GLSL_ONCE \n"
                  "#define %s_GLSL_ONCE \n\n",
                  config->define.upper,
                  config->define.upper);

          fprintf(target->state->header, "#define %s\n\n", config->define.upper);

          fprintf(
            target->state->header,
            "#define HS_SLAB_THREADS_LOG2    %u                              \n"
            "#define HS_SLAB_THREADS         (1 << HS_SLAB_THREADS_LOG2)     \n"
            "#define HS_SLAB_WIDTH_LOG2      %u                              \n"
            "#define HS_SLAB_WIDTH           (1 << HS_SLAB_WIDTH_LOG2)       \n"
            "#define HS_SLAB_HEIGHT          %u                              \n"
            "#define HS_SLAB_KEYS            (HS_SLAB_WIDTH * HS_SLAB_HEIGHT)\n"
            "#define HS_REG_LAST(c)          c##%u                           \n"
            "#define HS_KEY_DWORDS           %u                              \n"
            "#define HS_VAL_DWORDS           0                               \n"
            "#define HS_BS_SLABS             %u                              \n"
            "#define HS_BS_SLABS_LOG2_RU     %u                              \n"
            "#define HS_BC_SLABS_LOG2_MAX    %u                              \n"
            "#define HS_FM_BLOCK_HEIGHT      %u                              \n"
            "#define HS_FM_SCALE_MIN         %u                              \n"
            "#define HS_FM_SCALE_MAX         %u                              \n"
            "#define HS_HM_BLOCK_HEIGHT      %u                              \n"
            "#define HS_HM_SCALE_MIN         %u                              \n"
            "#define HS_HM_SCALE_MAX         %u                              \n"
            "#define HS_EMPTY                                                \n"
            "#define HS_KV_IN_SET            %u                              \n"
            "#define HS_KV_IN_BINDING        %u                              \n"
            "#define HS_KV_OUT_SET           %u                              \n"
            "#define HS_KV_OUT_BINDING       %u                              \n"
            "#define HS_IS_IN_PLACE          ((HS_KV_IN_SET == HS_KV_OUT_SET) && (HS_KV_IN_BINDING == HS_KV_OUT_BINDING))\n\n",
            config->warp.lanes_log2,  // NOTE: the number of slab threads might
            config->warp.lanes_log2,  // NOTE: not always equal slab width
            config->thread.regs,
            config->thread.regs,
            config->type.dwords,
            merge->warps,
            msb_idx_u32(pow2_ru_u32(merge->warps)),
            bc_max,
            config->merge.flip.warps,
            config->merge.flip.lo,
            config->merge.flip.hi,
            config->merge.half.warps,
            config->merge.half.lo,
            config->merge.half.hi,
            config->glsl.in.set,
            config->glsl.in.binding,
            config->glsl.out.set,
            config->glsl.out.binding);

          fprintf(target->state->header, "#define HS_SLAB_ROWS()    \\\n");

          for (uint32_t ii = 1; ii <= config->thread.regs; ii++)
            fprintf(target->state->header, "  HS_SLAB_ROW( %3u, %3u ) \\\n", ii, ii - 1);

          fprintf(target->state->header,
                  "  HS_EMPTY\n"
                  "          \n");

          fprintf(target->state->header, "#define HS_TRANSPOSE_SLAB()                \\\n");

          for (uint32_t ii = 1; ii <= config->warp.lanes_log2; ii++)
            fprintf(target->state->header, "  HS_TRANSPOSE_STAGE( %u )                  \\\n", ii);

          struct hsg_transpose_state state[1] = {
            {.header = target->state->header, .config = config}};

          hsg_transpose(config->warp.lanes_log2,
                        config->thread.regs,
                        hsg_transpose_blend,
                        state,
                        hsg_transpose_remap,
                        state);

          fprintf(target->state->header,
                  "  HS_EMPTY\n"
                  "          \n");
        }
        break;

      case HSG_OP_TYPE_TARGET_END:
        // decorate the files
        fprintf(target->state->header,
                "#endif \n"
                "       \n"
                "//     \n"
                "//     \n"
                "//     \n"
                "       \n");

        // close files
        fclose(target->state->header);
        fclose(target->state->modules);

        // free state
        free(target->state);
        break;

        case HSG_OP_TYPE_FILL_IN_KERNEL_PROTO: {
          fprintf(target->state->modules, "hs_fill_in\n");

          target->state->source = fopen("comp/hs_fill_in.comp", "wb");

          hsg_copyright(target->state->source);

          hsg_macros(config, target->state->source);

          fprintf(target->state->source, "HS_FILL_IN_KERNEL_PROTO()\n");
        }
        break;

        case HSG_OP_TYPE_FILL_IN_KERNEL_BODY: {
          fprintf(target->state->source, "HS_FILL_IN_BODY()\n");
        }
        break;

        case HSG_OP_TYPE_FILL_OUT_KERNEL_PROTO: {
          fprintf(target->state->modules, "hs_fill_out\n");

          target->state->source = fopen("comp/hs_fill_out.comp", "wb");

          hsg_copyright(target->state->source);

          hsg_macros(config, target->state->source);

          fprintf(target->state->source, "HS_FILL_OUT_KERNEL_PROTO()\n");
        }
        break;

        case HSG_OP_TYPE_FILL_OUT_KERNEL_BODY: {
          fprintf(target->state->source, "HS_FILL_OUT_BODY()\n");
        }
        break;

        case HSG_OP_TYPE_TRANSPOSE_KERNEL_PROTO: {
          fprintf(target->state->modules, "hs_transpose\n");

          target->state->source = fopen("comp/hs_transpose.comp", "wb");

          hsg_copyright(target->state->source);

          hsg_macros(config, target->state->source);

          fprintf(target->state->source, "HS_TRANSPOSE_KERNEL_PROTO()\n");
        }
        break;

        case HSG_OP_TYPE_TRANSPOSE_KERNEL_PREAMBLE: {
          fprintf(target->state->source, "HS_SUBGROUP_PREAMBLE();\n");

          fprintf(target->state->source, "HS_SLAB_GLOBAL_IDX_OUT();\n");
        }
        break;

        case HSG_OP_TYPE_TRANSPOSE_KERNEL_BODY: {
          fprintf(target->state->source, "HS_TRANSPOSE_SLAB()\n");
        }
        break;

        case HSG_OP_TYPE_BS_KERNEL_PROTO: {
          struct hsg_merge const * const m = merge + ops->a;

          uint32_t const bs  = pow2_ru_u32(m->warps);
          uint32_t const msb = msb_idx_u32(bs);

          fprintf(target->state->modules, "hs_bs_%u\n", msb);

          char filename[] = {"comp/hs_bs_XX.comp"};
          sprintf(filename, "comp/hs_bs_%u.comp", msb);

          target->state->source = fopen(filename, "wb");

          hsg_copyright(target->state->source);

          hsg_macros(config, target->state->source);

          if (m->warps > 1)
            {
              fprintf(target->state->source,
                      "HS_BLOCK_LOCAL_MEM_DECL(%u,%u);\n\n",
                      m->warps * config->warp.lanes,
                      m->rows_bs);
            }

          fprintf(target->state->source, "HS_BS_KERNEL_PROTO(%u,%u)\n", m->warps, msb);
        }
        break;

        case HSG_OP_TYPE_BS_KERNEL_PREAMBLE: {
          fprintf(target->state->source, "HS_SUBGROUP_PREAMBLE();\n");

          fprintf(target->state->source, "HS_SLAB_GLOBAL_IDX_IN_OUT();\n");
        }
        break;

        case HSG_OP_TYPE_BC_KERNEL_PROTO: {
          struct hsg_merge const * const m = merge + ops->a;

          uint32_t const msb = msb_idx_u32(m->warps);

          fprintf(target->state->modules, "hs_bc_%u\n", msb);

          char filename[] = {"comp/hs_bc_XX.comp"};
          sprintf(filename, "comp/hs_bc_%u.comp", msb);

          target->state->source = fopen(filename, "wb");

          hsg_copyright(target->state->source);

          hsg_macros(config, target->state->source);

          if (m->warps > 1)
            {
              fprintf(target->state->source,
                      "HS_BLOCK_LOCAL_MEM_DECL(%u,%u);\n\n",
                      m->warps * config->warp.lanes,
                      m->rows_bc);
            }

          fprintf(target->state->source, "HS_BC_KERNEL_PROTO(%u,%u)\n", m->warps, msb);
        }
        break;

        case HSG_OP_TYPE_BC_KERNEL_PREAMBLE: {
          fprintf(target->state->source, "HS_SUBGROUP_PREAMBLE()\n");

          fprintf(target->state->source, "HS_SLAB_GLOBAL_IDX_OUT();\n");
        }
        break;

        case HSG_OP_TYPE_FM_KERNEL_PROTO: {
          fprintf(target->state->modules, "hs_fm_%u_%u\n", ops->a, ops->b);

          char filename[] = {"comp/hs_fm_X_XX.comp"};
          sprintf(filename, "comp/hs_fm_%u_%u.comp", ops->a, ops->b);

          target->state->source = fopen(filename, "wb");

          hsg_copyright(target->state->source);

          hsg_macros(config, target->state->source);

          fprintf(target->state->source, "HS_FM_KERNEL_PROTO(%u,%u)\n", ops->a, ops->b);
        }
        break;

        case HSG_OP_TYPE_FM_KERNEL_PREAMBLE: {
          fprintf(target->state->source, "HS_SUBGROUP_PREAMBLE()\n");

          fprintf(target->state->source, "HS_FM_PREAMBLE(%u);\n", ops->a);
        }
        break;

        case HSG_OP_TYPE_HM_KERNEL_PROTO: {
          fprintf(target->state->modules, "hs_hm_%u\n", ops->a);

          char filename[] = {"comp/hs_hm_X.comp"};
          sprintf(filename, "comp/hs_hm_%u.comp", ops->a);

          target->state->source = fopen(filename, "wb");

          hsg_copyright(target->state->source);

          hsg_macros(config, target->state->source);

          fprintf(target->state->source, "HS_HM_KERNEL_PROTO(%u)\n", ops->a);
        }
        break;

        case HSG_OP_TYPE_HM_KERNEL_PREAMBLE: {
          fprintf(target->state->source, "HS_SUBGROUP_PREAMBLE()\n");

          fprintf(target->state->source, "HS_HM_PREAMBLE(%u);\n", ops->a);
        }
        break;

        case HSG_OP_TYPE_BX_REG_GLOBAL_LOAD: {
          static char const * const sgl_str[] = {
            "HS_SLAB_GLOBAL_LOAD_IN",
            "HS_SLAB_GLOBAL_LOAD_OUT",
          };

          fprintf(target->state->source,
                  "HS_KEY_TYPE r%-3u = %s(%u);\n",
                  ops->n,
                  sgl_str[ops->v],
                  ops->n - 1);
        }
        break;

      case HSG_OP_TYPE_BX_REG_GLOBAL_STORE:
        fprintf(target->state->source, "HS_SLAB_GLOBAL_STORE_OUT(%u,r%u);\n", ops->n - 1, ops->n);
        break;

      case HSG_OP_TYPE_HM_REG_GLOBAL_LOAD:
        fprintf(target->state->source,
                "HS_KEY_TYPE r%-3u = HS_XM_GLOBAL_LOAD_L(%u);\n",
                ops->a,
                ops->b);
        break;

      case HSG_OP_TYPE_HM_REG_GLOBAL_STORE:
        fprintf(target->state->source, "HS_XM_GLOBAL_STORE_L(%-3u,r%u);\n", ops->b, ops->a);
        break;

      case HSG_OP_TYPE_FM_REG_GLOBAL_LOAD_LEFT:
        fprintf(target->state->source,
                "HS_KEY_TYPE r%-3u = HS_XM_GLOBAL_LOAD_L(%u);\n",
                ops->a,
                ops->b);
        break;

      case HSG_OP_TYPE_FM_REG_GLOBAL_STORE_LEFT:
        fprintf(target->state->source, "HS_XM_GLOBAL_STORE_L(%-3u,r%u);\n", ops->b, ops->a);
        break;

      case HSG_OP_TYPE_FM_REG_GLOBAL_LOAD_RIGHT:
        fprintf(target->state->source,
                "HS_KEY_TYPE r%-3u = HS_FM_GLOBAL_LOAD_R(%u);\n",
                ops->b,
                ops->a);
        break;

      case HSG_OP_TYPE_FM_REG_GLOBAL_STORE_RIGHT:
        fprintf(target->state->source, "HS_FM_GLOBAL_STORE_R(%-3u,r%u);\n", ops->a, ops->b);
        break;

        case HSG_OP_TYPE_FM_MERGE_RIGHT_PRED: {
          if (ops->a <= ops->b)
            {
              fprintf(target->state->source, "if (HS_FM_IS_NOT_LAST_SPAN() || (fm_frac == 0))\n");
            }
          else if (ops->b > 1)
            {
              fprintf(target->state->source, "else if (fm_frac == %u)\n", ops->b);
            }
          else
            {
              fprintf(target->state->source, "else\n");
            }
        }
        break;

      case HSG_OP_TYPE_SLAB_FLIP:
        fprintf(target->state->source, "HS_SLAB_FLIP_PREAMBLE(%u);\n", ops->n - 1);
        break;

      case HSG_OP_TYPE_SLAB_HALF:
        fprintf(target->state->source, "HS_SLAB_HALF_PREAMBLE(%u);\n", ops->n / 2);
        break;

      case HSG_OP_TYPE_CMP_FLIP:
        fprintf(target->state->source, "HS_CMP_FLIP(%-3u,r%-3u,r%-3u);\n", ops->a, ops->b, ops->c);
        break;

      case HSG_OP_TYPE_CMP_HALF:
        fprintf(target->state->source, "HS_CMP_HALF(%-3u,r%-3u);\n", ops->a, ops->b);
        break;

      case HSG_OP_TYPE_CMP_XCHG:
        if (ops->c == UINT32_MAX)
          {
            fprintf(target->state->source, "HS_CMP_XCHG(r%-3u,r%-3u);\n", ops->a, ops->b);
          }
        else
          {
            fprintf(target->state->source,
                    "HS_CMP_XCHG(r%u_%u,r%u_%u);\n",
                    ops->c,
                    ops->a,
                    ops->c,
                    ops->b);
          }
        break;

      case HSG_OP_TYPE_BS_REG_SHARED_STORE_V:
        fprintf(target->state->source,
                "HS_BX_LOCAL_V(%-3u * HS_SLAB_THREADS * %-3u) = r%u;\n",
                merge[ops->a].warps,
                ops->c,
                ops->b);
        break;

      case HSG_OP_TYPE_BS_REG_SHARED_LOAD_V:
        fprintf(target->state->source,
                "r%-3u = HS_BX_LOCAL_V(%-3u * HS_SLAB_THREADS * %-3u);\n",
                ops->b,
                merge[ops->a].warps,
                ops->c);
        break;

      case HSG_OP_TYPE_BC_REG_SHARED_LOAD_V:
        fprintf(target->state->source,
                "HS_KEY_TYPE r%-3u = HS_BX_LOCAL_V(%-3u * HS_SLAB_THREADS * %-3u);\n",
                ops->b,
                ops->a,
                ops->c);
        break;

      case HSG_OP_TYPE_BX_REG_SHARED_STORE_LEFT:
        fprintf(target->state->source,
                "HS_SLAB_LOCAL_L(%5u) = r%u_%u;\n",
                ops->b * config->warp.lanes,
                ops->c,
                ops->a);
        break;

      case HSG_OP_TYPE_BS_REG_SHARED_STORE_RIGHT:
        fprintf(target->state->source,
                "HS_SLAB_LOCAL_R(%5u) = r%u_%u;\n",
                ops->b * config->warp.lanes,
                ops->c,
                ops->a);
        break;

      case HSG_OP_TYPE_BS_REG_SHARED_LOAD_LEFT:
        fprintf(target->state->source,
                "HS_KEY_TYPE r%u_%-3u = HS_SLAB_LOCAL_L(%u);\n",
                ops->c,
                ops->a,
                ops->b * config->warp.lanes);
        break;

      case HSG_OP_TYPE_BS_REG_SHARED_LOAD_RIGHT:
        fprintf(target->state->source,
                "HS_KEY_TYPE r%u_%-3u = HS_SLAB_LOCAL_R(%u);\n",
                ops->c,
                ops->a,
                ops->b * config->warp.lanes);
        break;

      case HSG_OP_TYPE_BC_REG_GLOBAL_LOAD_LEFT:
        fprintf(target->state->source,
                "HS_KEY_TYPE r%u_%-3u = HS_BC_GLOBAL_LOAD_L(%u);\n",
                ops->c,
                ops->a,
                ops->b);
        break;

      case HSG_OP_TYPE_BLOCK_SYNC:
        fprintf(target->state->source, "HS_BLOCK_BARRIER();\n");
        //
        // FIXME - Named barriers to allow coordinating warps to proceed?
        //
        break;

        case HSG_OP_TYPE_BS_FRAC_PRED: {
          if (ops->m == 0)
            {
              fprintf(target->state->source, "if (warp_idx < bs_full)\n");
            }
          else
            {
              fprintf(target->state->source, "else if (bs_frac == %u)\n", ops->w);
            }
        }
        break;

        case HSG_OP_TYPE_BS_MERGE_H_PREAMBLE: {
          struct hsg_merge const * const m = merge + ops->a;

          fprintf(target->state->source, "HS_BS_MERGE_H_PREAMBLE(%u);\n", m->warps);
        }
        break;

        case HSG_OP_TYPE_BC_MERGE_H_PREAMBLE: {
          struct hsg_merge const * const m = merge + ops->a;

          fprintf(target->state->source, "HS_BC_MERGE_H_PREAMBLE(%u);\n", m->warps);
        }
        break;

      case HSG_OP_TYPE_BX_MERGE_H_PRED:
        fprintf(target->state->source, "if (HS_SUBGROUP_ID() < %u)\n", ops->a);
        break;

        case HSG_OP_TYPE_BS_ACTIVE_PRED: {
          struct hsg_merge const * const m = merge + ops->a;

          if (m->warps <= 32)
            {
              fprintf(target->state->source,
                      "if (((1u << HS_SUBGROUP_ID()) & 0x%08X) != 0)\n",
                      m->levels[ops->b].active.b32a2[0]);
            }
          else
            {
              fprintf(target->state->source,
                      "if (((1UL << HS_SUBGROUP_ID()) & 0x%08X%08XL) != 0L)\n",
                      m->levels[ops->b].active.b32a2[1],
                      m->levels[ops->b].active.b32a2[0]);
            }
        }
        break;

      default:
        fprintf(stderr, "type not found: %s\n", hsg_op_type_string[ops->type]);
        exit(EXIT_FAILURE);
        break;
    }
}

//
//
//
