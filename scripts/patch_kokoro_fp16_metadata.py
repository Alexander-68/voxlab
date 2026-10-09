#!/usr/bin/env python3
"""
Utility script to inspect and copy Sherpa-ONNX metadata properties from a reference
Kokoro ONNX model (e.g. model.onnx) to a target ONNX model (e.g. model.fp16.onnx).

Raw ONNX checkpoints downloaded from HuggingFace (such as hexgrad/Kokoro-82M or kokoro-onnx)
often lack runtime metadata properties required by the Sherpa-ONNX offline TTS engine:
  - sample_rate (e.g. 24000)
  - model_type (kokoro)
  - version (e.g. 2)
  - has_espeak (1)
  - style_dim (510,1,256)
  - n_speakers (e.g. 54 or 103)
  - id2speaker / speaker2id / speaker_names

This tool ensures the target model contains all metadata properties so it can be loaded
seamlessly by sherpa-onnx-offline-tts and VoxLab.

Usage:
    python scripts/patch_kokoro_fp16_metadata.py
    python scripts/patch_kokoro_fp16_metadata.py --target models/kokoro-multi-lang-v1_0/model.fp16.onnx --ref models/kokoro-multi-lang-v1_0/model.onnx
    python scripts/patch_kokoro_fp16_metadata.py --check models/kokoro-multi-lang-v1_0/model.fp16.onnx
"""

import argparse
import os
import sys

try:
    import onnx
except ImportError:
    print("[Error] The 'onnx' Python package is required. Install it using: pip install onnx")
    sys.exit(1)


def inspect_metadata(model_path: str):
    if not os.path.exists(model_path):
        print(f"[Error] File not found: {model_path}")
        return False

    print(f"Inspecting ONNX metadata for: {model_path}")
    m = onnx.load(model_path, load_external_data=False)
    if not m.metadata_props:
        print("  (No metadata properties found)")
        return True

    for p in m.metadata_props:
        val_preview = p.value if len(p.value) <= 80 else p.value[:77] + "..."
        print(f"  {p.key} = {val_preview}")
    return True


def patch_metadata(target_path: str, reference_path: str) -> bool:
    if not os.path.exists(target_path):
        print(f"[Error] Target model not found: {target_path}")
        return False
    if not os.path.exists(reference_path):
        print(f"[Error] Reference model not found: {reference_path}")
        return False

    print(f"[1/3] Loading target model: {target_path}")
    m_target = onnx.load(target_path)

    print(f"[2/3] Loading reference model metadata: {reference_path}")
    m_ref = onnx.load(reference_path, load_external_data=False)

    existing_keys = {p.key for p in m_target.metadata_props}
    added_keys = []
    for p in m_ref.metadata_props:
        if p.key not in existing_keys:
            m_target.metadata_props.append(p)
            added_keys.append(p.key)

    if added_keys:
        print(f"[3/3] Injected missing metadata keys ({len(added_keys)}): {', '.join(added_keys)}")
        onnx.save(m_target, target_path)
        print(f"[OK] Successfully patched and saved: {target_path}")
    else:
        print("[OK] Target model already contains all required Sherpa-ONNX metadata properties.")

    return True


def main():
    parser = argparse.ArgumentParser(
        description="Inject or verify Sherpa-ONNX metadata properties on Kokoro ONNX models."
    )
    parser.add_argument(
        "target",
        nargs="?",
        default="models/kokoro-multi-lang-v1_0/model.fp16.onnx",
        help="Path to target model (default: models/kokoro-multi-lang-v1_0/model.fp16.onnx)",
    )
    parser.add_argument(
        "ref",
        nargs="?",
        default="models/kokoro-multi-lang-v1_0/model.onnx",
        help="Path to reference model with valid metadata (default: models/kokoro-multi-lang-v1_0/model.onnx)",
    )
    parser.add_argument(
        "--check",
        action="store_true",
        help="Inspect and print metadata of the target model without modifying it",
    )

    args = parser.parse_args()

    if args.check:
        success = inspect_metadata(args.target)
        sys.exit(0 if success else 1)

    success = patch_metadata(args.target, args.ref)
    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()
