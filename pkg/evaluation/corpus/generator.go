package corpus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"voxlab/pkg/audio"
	"voxlab/pkg/engine"
)

// BaseUtterance defines a unique Kokoro synthesis candidate.
type BaseUtterance struct {
	BaseID       string
	Description  string
	Mode         TestCaseMode
	Text         string
	Voice        string
	SpeakingRate float64
	Accent       string
	Gender       string
	Category     string
	Expected     string
	Lang         string // engine.TTSRequest.Lang
	AltText      string // EvalItem.AltText
}

// MatrixVariation defines a perturbation variant derived from a base utterance.
type MatrixVariation struct {
	Suffix      string
	Description string
	NoiseType   string
	SNR         *float64
	Expected    string // If empty, inherits BaseUtterance.Expected
}

// DefaultBaseUtterances returns the 16 curated base speech phrases spanning 12+ voices and 6 speaking rates.
func DefaultBaseUtterances() []BaseUtterance {
	return []BaseUtterance{
		{
			BaseID:       "CMD_SETTINGS_AF",
			Description:  "Open Settings (US Female, Normal)",
			Mode:         ModeCommand,
			Text:         "open settings",
			Voice:        "af_heart",
			SpeakingRate: 1.0,
			Accent:       "en-US",
			Gender:       "female",
			Category:     "command",
			Expected:     "NAV_SETTINGS",
		},
		{
			BaseID:       "CMD_SETTINGS_BM",
			Description:  "Open Settings (UK Male, Fast 1.2x)",
			Mode:         ModeCommand,
			Text:         "open settings",
			Voice:        "bm_george",
			SpeakingRate: 1.2,
			Accent:       "en-GB",
			Gender:       "male",
			Category:     "command",
			Expected:     "NAV_SETTINGS",
		},
		{
			BaseID:       "CMD_START_REC_AM",
			Description:  "Start Recording (US Male, Normal)",
			Mode:         ModeCommand,
			Text:         "start recording",
			Voice:        "am_adam",
			SpeakingRate: 1.0,
			Accent:       "en-US",
			Gender:       "male",
			Category:     "command",
			Expected:     "START_RECORDING",
		},
		{
			BaseID:       "CMD_START_REC_BF",
			Description:  "Start Recording (UK Female, Brisk 1.3x)",
			Mode:         ModeCommand,
			Text:         "start recording now",
			Voice:        "bf_emma",
			SpeakingRate: 1.3,
			Accent:       "en-GB",
			Gender:       "female",
			Category:     "command",
			Expected:     "START_RECORDING",
		},
		{
			BaseID:       "CMD_STOP_REC_AF",
			Description:  "Stop Recording (US Female)",
			Mode:         ModeCommand,
			Text:         "stop recording",
			Voice:        "af_bella",
			SpeakingRate: 1.0,
			Accent:       "en-US",
			Gender:       "female",
			Category:     "command",
			Expected:     "STOP_RECORDING",
		},
		{
			BaseID:       "CMD_DIM_BM",
			Description:  "Dim Screen (UK Male Narrator)",
			Mode:         ModeCommand,
			Text:         "dim screen",
			Voice:        "bm_lewis",
			SpeakingRate: 1.0,
			Accent:       "en-GB",
			Gender:       "male",
			Category:     "command",
			Expected:     "DISPLAY_DARK",
		},
		{
			BaseID:       "CMD_SCOPE_AF",
			Description:  "Show Endoscope on Main Monitor (US Female)",
			Mode:         ModeCommand,
			Text:         "show endoscope on the main monitor",
			Voice:        "af_nicole",
			SpeakingRate: 1.0,
			Accent:       "en-US",
			Gender:       "female",
			Category:     "command",
			Expected:     "ROUTE_VIDEO",
		},
		{
			BaseID:       "CMD_PREF_AM",
			Description:  "Open System Preferences (US Male, Deliberate 0.8x)",
			Mode:         ModeCommand,
			Text:         "open system preferences",
			Voice:        "am_adam",
			SpeakingRate: 0.8,
			Accent:       "en-US",
			Gender:       "male",
			Category:     "command",
			Expected:     "NAV_SETTINGS",
		},
		{
			BaseID:       "DICT_BENCH_BF",
			Description:  "Dictation: Voice Test Bench (UK Female)",
			Mode:         ModeDictation,
			Text:         "The voice pipeline test bench is operational and ready.",
			Voice:        "bf_emma",
			SpeakingRate: 1.0,
			Accent:       "en-GB",
			Gender:       "female",
			Category:     "dictation",
			Expected:     "pipeline test bench",
		},
		{
			BaseID:       "DICT_VITALS_AM",
			Description:  "Dictation: Patient Vitals (US Male)",
			Mode:         ModeDictation,
			Text:         "Patient vital signs are stable with blood pressure one twenty over eighty.",
			AltText:      "Patient vital signs are stable with blood pressure 120 over 80.",
			Voice:        "am_michael",
			SpeakingRate: 1.0,
			Accent:       "en-US",
			Gender:       "male",
			Category:     "dictation",
			Expected:     "blood pressure",
		},
		{
			BaseID:       "DICT_INCISION_AF",
			Description:  "Dictation: Surgical Findings (US Female, 1.1x)",
			Mode:         ModeDictation,
			Text:         "Incision site clean with no signs of inflammation or erythema.",
			Voice:        "af_sky",
			SpeakingRate: 1.1,
			Accent:       "en-US",
			Gender:       "female",
			Category:     "dictation",
			Expected:     "incision site",
		},
		{
			BaseID:       "DICT_MED_BF",
			Description:  "Dictation: Medication Dosage (UK Female, 0.9x)",
			Mode:         ModeDictation,
			Text:         "Administer two milligrams of midazolam intravenously.",
			AltText:      "Administer 2 mg of midazolam intravenously.",
			Voice:        "bf_isabella",
			SpeakingRate: 0.9,
			Accent:       "en-GB",
			Gender:       "female",
			Category:     "dictation",
			Expected:     "intravenously",
		},
		{
			BaseID:       "NEG_STOP_AF",
			Description:  "Negation: Do Not Stop Recording (US Female)",
			Mode:         ModeCommand,
			Text:         "do not stop recording",
			Voice:        "af_heart",
			SpeakingRate: 1.0,
			Accent:       "en-US",
			Gender:       "female",
			Category:     "negation",
			Expected:     "REJECT:NEGATION_DETECTED",
		},
		{
			BaseID:       "NEG_SETTINGS_AM",
			Description:  "Negation: Do Not Open Settings (US Male, 1.1x)",
			Mode:         ModeCommand,
			Text:         "do not open settings during procedure",
			Voice:        "am_adam",
			SpeakingRate: 1.1,
			Accent:       "en-US",
			Gender:       "male",
			Category:     "negation",
			Expected:     "REJECT:NEGATION_DETECTED",
		},
		{
			BaseID:       "DIST_LUNCH_AM",
			Description:  "Distractor: Lunch Time Chatter (US Male)",
			Mode:         ModeCommand,
			Text:         "what time is lunch scheduled for today",
			Voice:        "am_echo",
			SpeakingRate: 1.0,
			Accent:       "en-US",
			Gender:       "male",
			Category:     "distractor",
			Expected:     "REJECT:OUT_OF_DOMAIN_CHATTER",
		},
		{
			BaseID:       "CMD_ES_EF",
			Description:  "Spanish Accented Command: Open Settings",
			Mode:         ModeCommand,
			Text:         "open settings",
			Voice:        "ef_dora",
			Lang:         "en", // English read with a Spanish voice, the point of this item
			SpeakingRate: 1.0,
			Accent:       "es",
			Gender:       "female",
			Category:     "command",
			Expected:     "NAV_SETTINGS",
		},
		// --- Multilingual Dictation Test Cases ---
		// Spanish (es-ES) Dictations
		{
			BaseID:       "DICT_ES_VITALS_EF",
			Description:  "Dictation: Patient Vitals (Spanish Female)",
			Mode:         ModeDictation,
			Text:         "Los signos vitales del paciente son estables con presión arterial normal.",
			Voice:        "ef_dora",
			SpeakingRate: 1.0,
			Accent:       "es-ES",
			Gender:       "female",
			Category:     "dictation",
			Expected:     "signos vitales",
		},
		{
			BaseID:       "DICT_ES_INCISION_AM",
			Description:  "Dictation: Surgical Incision (Spanish Male)",
			Mode:         ModeDictation,
			Text:         "El sitio de la incisión está limpio sin signos de inflamación ni eritema.",
			Voice:        "em_alex",
			SpeakingRate: 1.0,
			Accent:       "es-ES",
			Gender:       "male",
			Category:     "dictation",
			Expected:     "sitio de la incisión",
		},
		{
			BaseID:       "DICT_ES_MED_ES",
			Description:  "Dictation: Medication Dosage (Spanish Male Santa)",
			Mode:         ModeDictation,
			Text:         "Administrar dos miligramos de midazolam por vía intravenosa.",
			AltText:      "Administrar 2 mg de midazolam por vía intravenosa.",
			Voice:        "em_santa",
			SpeakingRate: 1.0,
			Accent:       "es-ES",
			Gender:       "male",
			Category:     "dictation",
			Expected:     "midazolam",
		},
		{
			BaseID:       "DICT_ES_BENCH_EF",
			Description:  "Dictation: Voice Test Bench Status (Spanish Female, 1.1x)",
			Mode:         ModeDictation,
			Text:         "El banco de pruebas de la tubería de voz está operativo y listo.",
			Voice:        "ef_dora",
			SpeakingRate: 1.1,
			Accent:       "es-ES",
			Gender:       "female",
			Category:     "dictation",
			Expected:     "banco de pruebas",
		},

		// Brazilian Portuguese (pt-BR) Dictations
		{
			BaseID:       "DICT_PT_VITALS_PF",
			Description:  "Dictation: Patient Vitals (Portuguese Female)",
			Mode:         ModeDictation,
			Text:         "Os sinais vitais do paciente estão estáveis e a pressão arterial está normal.",
			Voice:        "pf_dora",
			SpeakingRate: 1.0,
			Accent:       "pt-BR",
			Gender:       "female",
			Category:     "dictation",
			Expected:     "sinais vitais",
		},
		{
			BaseID:       "DICT_PT_INCISION_PM",
			Description:  "Dictation: Surgical Findings (Portuguese Male Alex)",
			Mode:         ModeDictation,
			Text:         "A incisão cirúrgica está limpa e sem sinais de infecção ou edema.",
			Voice:        "pm_alex",
			SpeakingRate: 1.0,
			Accent:       "pt-BR",
			Gender:       "male",
			Category:     "dictation",
			Expected:     "incisão cirúrgica",
		},
		{
			BaseID:       "DICT_PT_MED_PS",
			Description:  "Dictation: Medication Dosage (Portuguese Male Santa)",
			Mode:         ModeDictation,
			Text:         "Administrar dois miligramas de midazolam por via intravenosa.",
			AltText:      "Administrar 2 mg de midazolam por via intravenosa.",
			Voice:        "pm_santa",
			SpeakingRate: 1.0,
			Accent:       "pt-BR",
			Gender:       "male",
			Category:     "dictation",
			Expected:     "midazolam",
		},
		{
			BaseID:       "DICT_PT_STATUS_PF",
			Description:  "Dictation: System Status (Portuguese Female, 1.1x)",
			Mode:         ModeDictation,
			Text:         "O sistema de processamento de voz está operando perfeitamente.",
			Voice:        "pf_dora",
			SpeakingRate: 1.1,
			Accent:       "pt-BR",
			Gender:       "female",
			Category:     "dictation",
			Expected:     "processamento de voz",
		},

		// French (fr-FR) Dictations
		{
			BaseID:       "DICT_FR_VITALS_FF",
			Description:  "Dictation: Patient Vitals (French Female)",
			Mode:         ModeDictation,
			Text:         "Les constantes vitales du patient sont stables et la tension artérielle est normale.",
			Voice:        "ff_siwis",
			SpeakingRate: 1.0,
			Accent:       "fr-FR",
			Gender:       "female",
			Category:     "dictation",
			Expected:     "constantes vitales",
		},
		{
			BaseID:       "DICT_FR_INCISION_FF",
			Description:  "Dictation: Surgical Findings (French Female)",
			Mode:         ModeDictation,
			Text:         "Le site incisionnel est propre sans aucun signe d'inflammation ni d'érythème.",
			Voice:        "ff_siwis",
			SpeakingRate: 1.0,
			Accent:       "fr-FR",
			Gender:       "female",
			Category:     "dictation",
			Expected:     "site incisionnel",
		},
		{
			BaseID:       "DICT_FR_MED_FF",
			Description:  "Dictation: Medication Dosage (French Female)",
			Mode:         ModeDictation,
			Text:         "Administrer deux milligrammes de midazolam par voie intraveineuse.",
			AltText:      "Administrer 2 mg de midazolam par voie intraveineuse.",
			Voice:        "ff_siwis",
			SpeakingRate: 1.0,
			Accent:       "fr-FR",
			Gender:       "female",
			Category:     "dictation",
			Expected:     "midazolam",
		},
		{
			BaseID:       "DICT_FR_BENCH_FF",
			Description:  "Dictation: Test Bench Status (French Female, 1.1x)",
			Mode:         ModeDictation,
			Text:         "Le banc d'essai de reconnaissance vocale est opérationnel et prêt.",
			Voice:        "ff_siwis",
			SpeakingRate: 1.1,
			Accent:       "fr-FR",
			Gender:       "female",
			Category:     "dictation",
			Expected:     "banc d'essai",
		},

		// Italian (it-IT) Dictations
		{
			BaseID:       "DICT_IT_VITALS_IF",
			Description:  "Dictation: Patient Vitals (Italian Female)",
			Mode:         ModeDictation,
			Text:         "I parametri vitali del paziente sono stabili con pressione sanguigna nella norma.",
			Voice:        "if_sara",
			SpeakingRate: 1.0,
			Accent:       "it-IT",
			Gender:       "female",
			Category:     "dictation",
			Expected:     "parametri vitali",
		},
		{
			BaseID:       "DICT_IT_INCISION_IM",
			Description:  "Dictation: Surgical Findings (Italian Male)",
			Mode:         ModeDictation,
			Text:         "La ferita chirurgica è pulita senza segni di infiammazione o eritema.",
			Voice:        "im_nicola",
			SpeakingRate: 1.0,
			Accent:       "it-IT",
			Gender:       "male",
			Category:     "dictation",
			Expected:     "ferita chirurgica",
		},
		{
			BaseID:       "DICT_IT_MED_IF",
			Description:  "Dictation: Medication Dosage (Italian Female)",
			Mode:         ModeDictation,
			Text:         "Somministrare due milligrammi di midazolam per via endovenosa.",
			AltText:      "Somministrare 2 mg di midazolam per via endovenosa.",
			Voice:        "if_sara",
			SpeakingRate: 1.0,
			Accent:       "it-IT",
			Gender:       "female",
			Category:     "dictation",
			Expected:     "midazolam",
		},
		{
			BaseID:       "DICT_IT_BENCH_IM",
			Description:  "Dictation: Test Bench Status (Italian Male, 1.1x)",
			Mode:         ModeDictation,
			Text:         "Il sistema di riconoscimento vocale è pienamente operativo.",
			Voice:        "im_nicola",
			SpeakingRate: 1.1,
			Accent:       "it-IT",
			Gender:       "male",
			Category:     "dictation",
			Expected:     "riconoscimento vocale",
		},

		// Mandarin Chinese (zh-CN) Dictations (14 diverse clinical & technical samples)
		{
			BaseID:       "DICT_ZH_VITALS_XB",
			Description:  "Dictation: Patient Vitals (Mandarin Female Xiaobei)",
			Mode:         ModeDictation,
			Text:         "患者生命体征平稳，血压和心率均在正常范围内。",
			Voice:        "zf_xiaobei",
			SpeakingRate: 1.0,
			Accent:       "zh-CN",
			Gender:       "female",
			Category:     "dictation",
			Expected:     "生命体征",
		},
		{
			BaseID:       "DICT_ZH_INCISION_YJ",
			Description:  "Dictation: Surgical Incision (Mandarin Male Yunjian)",
			Mode:         ModeDictation,
			Text:         "手术切口干燥整洁，未见明显红肿渗出及感染迹象。",
			Voice:        "zm_yunjian",
			SpeakingRate: 1.0,
			Accent:       "zh-CN",
			Gender:       "male",
			Category:     "dictation",
			Expected:     "手术切口",
		},
		{
			BaseID:       "DICT_ZH_MED_XN",
			Description:  "Dictation: Medication Dosage (Mandarin Female Xiaoni)",
			Mode:         ModeDictation,
			Text:         "静脉推注咪达唑仑两毫克，密切观察患者镇静状态。",
			AltText:      "静脉推注咪达唑仑2毫克，密切观察患者镇静状态。",
			Voice:        "zf_xiaoni",
			SpeakingRate: 1.0,
			Accent:       "zh-CN",
			Gender:       "female",
			Category:     "dictation",
			Expected:     "咪达唑仑",
		},
		{
			BaseID:       "DICT_ZH_ANESTHESIA_YX",
			Description:  "Dictation: General Anesthesia (Mandarin Male Yunxi)",
			Mode:         ModeDictation,
			Text:         "全身麻醉诱导平稳，气管插管顺利，通气良好。",
			Voice:        "zm_yunxi",
			SpeakingRate: 1.0,
			Accent:       "zh-CN",
			Gender:       "male",
			Category:     "dictation",
			Expected:     "全身麻醉",
		},
		{
			BaseID:       "DICT_ZH_STATUS_XX",
			Description:  "Dictation: Pipeline Status (Mandarin Female Xiaoxiao)",
			Mode:         ModeDictation,
			Text:         "语音处理管道与音频测试平台已成功建立并就绪。",
			Voice:        "zf_xiaoxiao",
			SpeakingRate: 1.0,
			Accent:       "zh-CN",
			Gender:       "female",
			Category:     "dictation",
			Expected:     "语音处理",
		},
		{
			BaseID:       "DICT_ZH_POSTOP_XY",
			Description:  "Dictation: Postoperative Recovery (Mandarin Female Xiaoyi)",
			Mode:         ModeDictation,
			Text:         "术后神志清楚，自主呼吸恢复良好，安返病房。",
			Voice:        "zf_xiaoyi",
			SpeakingRate: 1.0,
			Accent:       "zh-CN",
			Gender:       "female",
			Category:     "dictation",
			Expected:     "自主呼吸",
		},
		{
			BaseID:       "DICT_ZH_BLOOD_YXIA",
			Description:  "Dictation: Intraoperative Bleeding (Mandarin Male Yunxia)",
			Mode:         ModeDictation,
			Text:         "术中出血量约五十毫升，已彻底止血，输液通畅。",
			Voice:        "zm_yunxia",
			SpeakingRate: 1.0,
			Accent:       "zh-CN",
			Gender:       "male",
			Category:     "dictation",
			Expected:     "出血量",
		},
		{
			BaseID:       "DICT_ZH_NEURO_YY",
			Description:  "Dictation: Neurological Examination (Mandarin Male Yunyang)",
			Mode:         ModeDictation,
			Text:         "神经反射正常，双侧瞳孔等大同圆，对光反射灵敏。",
			Voice:        "zm_yunyang",
			SpeakingRate: 1.0,
			Accent:       "zh-CN",
			Gender:       "male",
			Category:     "dictation",
			Expected:     "对光反射",
		},
		{
			BaseID:       "DICT_ZH_FAST_XB",
			Description:  "Dictation: Rapid Charting (Mandarin Female Xiaobei, 1.2x)",
			Mode:         ModeDictation,
			Text:         "快速记录：心律规整，呼吸平稳，双肺呼吸音清。",
			Voice:        "zf_xiaobei",
			SpeakingRate: 1.2,
			Accent:       "zh-CN",
			Gender:       "female",
			Category:     "dictation",
			Expected:     "呼吸平稳",
		},
		{
			BaseID:       "DICT_ZH_SLOW_YJ",
			Description:  "Dictation: Careful Clinical Note (Mandarin Male Yunjian, 0.9x)",
			Mode:         ModeDictation,
			Text:         "缓慢记录：建议持续心电监护，观察引流液性状。",
			Voice:        "zm_yunjian",
			SpeakingRate: 0.9,
			Accent:       "zh-CN",
			Gender:       "male",
			Category:     "dictation",
			Expected:     "心电监护",
		},
		{
			BaseID:       "DICT_ZH_DISCHARGE_XN",
			Description:  "Dictation: Discharge Instructions (Mandarin Female Xiaoni)",
			Mode:         ModeDictation,
			Text:         "遵医嘱规律服药，按时换药，一周后门诊复查。",
			Voice:        "zf_xiaoni",
			SpeakingRate: 1.0,
			Accent:       "zh-CN",
			Gender:       "female",
			Category:     "dictation",
			Expected:     "遵医嘱",
		},
		{
			BaseID:       "DICT_ZH_PATHOLOGY_YX",
			Description:  "Dictation: Intraoperative Pathology (Mandarin Male Yunxi)",
			Mode:         ModeDictation,
			Text:         "术中快速病理切片回报为良性病变，边缘未见残留。",
			Voice:        "zm_yunxi",
			SpeakingRate: 1.0,
			Accent:       "zh-CN",
			Gender:       "male",
			Category:     "dictation",
			Expected:     "病理切片",
		},
		{
			BaseID:       "DICT_ZH_OXYGEN_XX",
			Description:  "Dictation: Oxygen Therapy (Mandarin Female Xiaoxiao)",
			Mode:         ModeDictation,
			Text:         "鼻导管吸氧每分钟两升，经皮血氧饱和度百分之九十九。",
			Voice:        "zf_xiaoxiao",
			SpeakingRate: 1.0,
			Accent:       "zh-CN",
			Gender:       "female",
			Category:     "dictation",
			Expected:     "血氧饱和度",
		},
		{
			BaseID:       "DICT_ZH_DIAGNOSIS_XY",
			Description:  "Dictation: Preoperative Diagnosis (Mandarin Female Xiaoyi)",
			Mode:         ModeDictation,
			Text:         "初步诊断明确，无特殊禁忌证，拟行择期腹腔镜手术。",
			Voice:        "zf_xiaoyi",
			SpeakingRate: 1.0,
			Accent:       "zh-CN",
			Gender:       "female",
			Category:     "dictation",
			Expected:     "腹腔镜",
		},
	}
}

func ptrFloat(f float64) *float64 { return &f }

// BuildTier2MatrixDefinitions specifies the test combinations across the SNR ladder and perturbations.
func BuildTier2MatrixDefinitions() ([]BaseUtterance, map[string][]MatrixVariation) {
	bases := DefaultBaseUtterances()
	variations := make(map[string][]MatrixVariation)

	// Full SNR Ladder variations (+20, +15, +10, +5, 0, -5 dB)
	fullSNRLadder := []MatrixVariation{
		{Suffix: "SNR20", Description: "White Noise SNR +20 dB", NoiseType: audio.NoiseWhite, SNR: ptrFloat(20.0)},
		{Suffix: "SNR15_PINK", Description: "Pink Noise SNR +15 dB", NoiseType: audio.NoisePink, SNR: ptrFloat(15.0)},
		{Suffix: "SNR10", Description: "White Noise SNR +10 dB", NoiseType: audio.NoiseWhite, SNR: ptrFloat(10.0)},
		{Suffix: "SNR05", Description: "White Noise SNR +5 dB", NoiseType: audio.NoiseWhite, SNR: ptrFloat(5.0)},
		{Suffix: "SNR00", Description: "White Noise SNR 0 dB", NoiseType: audio.NoiseWhite, SNR: ptrFloat(0.0)},
		{Suffix: "SNRM05", Description: "White Noise SNR -5 dB", NoiseType: audio.NoiseWhite, SNR: ptrFloat(-5.0)},
	}

	// Mid SNR variations (+20, +10, +5, 0 dB)
	midSNRLadder := []MatrixVariation{
		{Suffix: "SNR20", Description: "White Noise SNR +20 dB", NoiseType: audio.NoiseWhite, SNR: ptrFloat(20.0)},
		{Suffix: "SNR10", Description: "White Noise SNR +10 dB", NoiseType: audio.NoiseWhite, SNR: ptrFloat(10.0)},
		{Suffix: "SNR05", Description: "White Noise SNR +5 dB", NoiseType: audio.NoiseWhite, SNR: ptrFloat(5.0)},
		{Suffix: "SNR00", Description: "White Noise SNR 0 dB", NoiseType: audio.NoiseWhite, SNR: ptrFloat(0.0)},
	}

	// Light SNR variations (+20, +10, +5 dB)
	lightSNRLadder := []MatrixVariation{
		{Suffix: "SNR20", Description: "White Noise SNR +20 dB", NoiseType: audio.NoiseWhite, SNR: ptrFloat(20.0)},
		{Suffix: "SNR10", Description: "White Noise SNR +10 dB", NoiseType: audio.NoiseWhite, SNR: ptrFloat(10.0)},
		{Suffix: "SNR05", Description: "White Noise SNR +5 dB", NoiseType: audio.NoiseWhite, SNR: ptrFloat(5.0)},
	}

	// Assign variations to representative base utterances
	variations["CMD_SETTINGS_AF"] = append(fullSNRLadder,
		MatrixVariation{Suffix: "TYPING_15", Description: "Typing Clacks SNR +15 dB", NoiseType: audio.NoiseTyping, SNR: ptrFloat(15.0)},
		MatrixVariation{Suffix: "TYPING_05", Description: "Typing Clacks SNR +5 dB", NoiseType: audio.NoiseTyping, SNR: ptrFloat(5.0)},
		MatrixVariation{Suffix: "BABBLE_15", Description: "Room Babble SNR +15 dB", NoiseType: audio.NoiseBabble, SNR: ptrFloat(15.0)},
		MatrixVariation{Suffix: "BABBLE_05", Description: "Room Babble SNR +5 dB", NoiseType: audio.NoiseBabble, SNR: ptrFloat(5.0)},
		MatrixVariation{Suffix: "REVERB", Description: "Room Reverberation (RT60 ~0.4s)", NoiseType: audio.NoiseReverb, SNR: ptrFloat(15.0)},
	)

	variations["CMD_START_REC_AM"] = append(fullSNRLadder,
		MatrixVariation{Suffix: "TYPING_15", Description: "Typing Clacks SNR +15 dB", NoiseType: audio.NoiseTyping, SNR: ptrFloat(15.0)},
		MatrixVariation{Suffix: "TYPING_05", Description: "Typing Clacks SNR +5 dB", NoiseType: audio.NoiseTyping, SNR: ptrFloat(5.0)},
		MatrixVariation{Suffix: "BABBLE_15", Description: "Room Babble SNR +15 dB", NoiseType: audio.NoiseBabble, SNR: ptrFloat(15.0)},
		MatrixVariation{Suffix: "BABBLE_05", Description: "Room Babble SNR +5 dB", NoiseType: audio.NoiseBabble, SNR: ptrFloat(5.0)},
		MatrixVariation{Suffix: "REVERB", Description: "Room Reverberation (RT60 ~0.4s)", NoiseType: audio.NoiseReverb, SNR: ptrFloat(15.0)},
	)

	variations["DICT_BENCH_BF"] = append(fullSNRLadder,
		MatrixVariation{Suffix: "TYPING_15", Description: "Typing Clacks SNR +15 dB", NoiseType: audio.NoiseTyping, SNR: ptrFloat(15.0)},
		MatrixVariation{Suffix: "TYPING_05", Description: "Typing Clacks SNR +5 dB", NoiseType: audio.NoiseTyping, SNR: ptrFloat(5.0)},
		MatrixVariation{Suffix: "BABBLE_15", Description: "Room Babble SNR +15 dB", NoiseType: audio.NoiseBabble, SNR: ptrFloat(15.0)},
		MatrixVariation{Suffix: "BABBLE_05", Description: "Room Babble SNR +5 dB", NoiseType: audio.NoiseBabble, SNR: ptrFloat(5.0)},
		MatrixVariation{Suffix: "REVERB", Description: "Room Reverberation (RT60 ~0.4s)", NoiseType: audio.NoiseReverb, SNR: ptrFloat(15.0)},
	)

	variations["DICT_VITALS_AM"] = append(midSNRLadder,
		MatrixVariation{Suffix: "TYPING_15", Description: "Typing Clacks SNR +15 dB", NoiseType: audio.NoiseTyping, SNR: ptrFloat(15.0)},
		MatrixVariation{Suffix: "TYPING_05", Description: "Typing Clacks SNR +5 dB", NoiseType: audio.NoiseTyping, SNR: ptrFloat(5.0)},
		MatrixVariation{Suffix: "REVERB", Description: "Room Reverberation (RT60 ~0.4s)", NoiseType: audio.NoiseReverb, SNR: ptrFloat(15.0)},
	)

	variations["DICT_INCISION_AF"] = []MatrixVariation{
		{Suffix: "BABBLE_15", Description: "Room Babble SNR +15 dB", NoiseType: audio.NoiseBabble, SNR: ptrFloat(15.0)},
		{Suffix: "BABBLE_05", Description: "Room Babble SNR +5 dB", NoiseType: audio.NoiseBabble, SNR: ptrFloat(5.0)},
	}

	variations["NEG_STOP_AF"] = lightSNRLadder
	variations["CMD_ES_EF"] = lightSNRLadder

	// Multilingual dictation acoustic variations
	variations["DICT_ES_VITALS_EF"] = []MatrixVariation{
		{Suffix: "SNR15_PINK", Description: "Pink Noise SNR +15 dB", NoiseType: audio.NoisePink, SNR: ptrFloat(15.0)},
	}
	variations["DICT_PT_VITALS_PF"] = []MatrixVariation{
		{Suffix: "SNR15_PINK", Description: "Pink Noise SNR +15 dB", NoiseType: audio.NoisePink, SNR: ptrFloat(15.0)},
	}
	variations["DICT_FR_VITALS_FF"] = []MatrixVariation{
		{Suffix: "SNR15_PINK", Description: "Pink Noise SNR +15 dB", NoiseType: audio.NoisePink, SNR: ptrFloat(15.0)},
	}
	variations["DICT_IT_VITALS_IF"] = []MatrixVariation{
		{Suffix: "SNR15_PINK", Description: "Pink Noise SNR +15 dB", NoiseType: audio.NoisePink, SNR: ptrFloat(15.0)},
	}
	variations["DICT_ZH_VITALS_XB"] = []MatrixVariation{
		{Suffix: "SNR15_PINK", Description: "Pink Noise SNR +15 dB", NoiseType: audio.NoisePink, SNR: ptrFloat(15.0)},
		{Suffix: "TYPING_15", Description: "Typing Clacks SNR +15 dB", NoiseType: audio.NoiseTyping, SNR: ptrFloat(15.0)},
	}
	variations["DICT_ZH_INCISION_YJ"] = []MatrixVariation{
		{Suffix: "BABBLE_15", Description: "Room Babble SNR +15 dB", NoiseType: audio.NoiseBabble, SNR: ptrFloat(15.0)},
		{Suffix: "REVERB", Description: "Room Reverberation (RT60 ~0.4s)", NoiseType: audio.NoiseReverb, SNR: ptrFloat(15.0)},
	}

	return bases, variations
}

// scaleToDBFS sets the level of the loudest 30 ms chunk, which is what the noise gate reacts to.
const chunkSamples = 480

func scaleToDBFS(samples []float32, dbfs float64) {
	var peak float64
	for i := 0; i < len(samples); i += chunkSamples {
		rms, _ := audio.CalculateRMS(samples[i:min(i+chunkSamples, len(samples))])
		peak = max(peak, rms)
	}
	gain := float32(math.Pow(10, dbfs/20) / peak)
	for i := range samples {
		samples[i] *= gain
	}
}

// LoadOrSynthesizeClip returns the 16 kHz clip at path, synthesizing and saving it first if missing.
// Existing clips are kept, so regenerating one means deleting it.
func LoadOrSynthesizeClip(path string, runner engine.SpeechEngine, req engine.TTSRequest) ([]float32, error) {
	if wavData, err := os.ReadFile(path); err == nil {
		if s, rate, err := audio.DecodeWAV(bytes.NewReader(wavData)); err == nil && len(s) > 0 {
			if rate != 16000 {
				s = audio.Resample(s, rate, 16000)
			}
			return s, nil
		}
	}
	if runner == nil {
		return nil, fmt.Errorf("speech runner required to synthesize %s", path)
	}
	log.Printf("   -> Synthesizing %s voice=%s rate=%.1f: %q", filepath.Base(path), req.Voice, req.Speed, req.Text)
	res, err := runner.Synthesize(req)
	if err != nil {
		return nil, fmt.Errorf("synthesis failed: %w", err)
	}

	resampled := audio.Resample(res.AudioSamples, 24000, 16000)
	samples := make([]float32, 16000*200/1000) // 200 ms lead
	samples = append(samples, resampled...)
	samples = append(samples, make([]float32, 16000)...) // 1 s trail

	wavBytes, err := audio.EncodeWAV(samples, 16000)
	if err != nil {
		return nil, fmt.Errorf("failed encoding %s: %w", path, err)
	}
	if err := os.WriteFile(path, wavBytes, 0644); err != nil {
		return nil, fmt.Errorf("failed writing %s: %w", path, err)
	}
	// Decode what was written, so perturbations match a later run that loads this clip from disk.
	samples, _, err = audio.DecodeWAV(bytes.NewReader(wavBytes))
	return samples, err
}

// GenerateTier2Corpus synthesizes base audio clips and creates the full acoustic degradation matrix.
func GenerateTier2Corpus(outDir string, runner engine.SpeechEngine) (*EvalManifest, error) {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("failed creating tier2 output directory: %w", err)
	}

	bases, variationsMap := BuildTier2MatrixDefinitions()

	manifest := &EvalManifest{
		Version: "2.0",
		Tier:    "tier-2-deep-matrix",
		Items:   make([]EvalItem, 0),
	}

	// Step 1: Synthesize base WAV files
	baseSamples := make(map[string][]float32)

	log.Printf("[Tier2 Generator] Synthesizing %d base utterances across Kokoro voices...", len(bases))
	for _, b := range bases {
		baseWavFilename := fmt.Sprintf("base_%s.wav", strings.ToLower(b.BaseID))
		samples16k, err := LoadOrSynthesizeClip(filepath.Join(outDir, baseWavFilename), runner, engine.TTSRequest{
			Text:       b.Text,
			Voice:      b.Voice,
			Speed:      b.SpeakingRate,
			SampleRate: 24000,
			Lang:       b.Lang,
		})
		if err != nil {
			return nil, fmt.Errorf("base utterance %s: %w", b.BaseID, err)
		}

		baseSamples[b.BaseID] = samples16k

		// Add clean base item to manifest
		dur := float64(len(samples16k)) / 16000.0
		manifest.Items = append(manifest.Items, EvalItem{
			ID:           fmt.Sprintf("T2_CLEAN_%s", b.BaseID),
			Description:  fmt.Sprintf("[Clean] %s", b.Description),
			Mode:         b.Mode,
			Text:         b.Text,
			AltText:      b.AltText,
			Voice:        b.Voice,
			SpeakingRate: b.SpeakingRate,
			Accent:       b.Accent,
			Category:     b.Category,
			NoiseType:    audio.NoiseClean,
			Expected:     b.Expected,
			WavFile:      baseWavFilename,
			DurationSec:  dur,
		})
	}

	// Step 2: Generate acoustic perturbations (SNR ladder, typing, babble, reverb)
	seed := int64(1001)
	for _, b := range bases {
		cleanSamples := baseSamples[b.BaseID]
		vars, hasVars := variationsMap[b.BaseID]
		if !hasVars {
			continue
		}

		for _, v := range vars {
			seed++
			perturbedSamples := audio.AddNoiseAtSNR(cleanSamples, v.NoiseType, *v.SNR, 16000, seed)
			filename := fmt.Sprintf("t2_%s_%s.wav", strings.ToLower(b.BaseID), strings.ToLower(v.Suffix))
			wavPath := filepath.Join(outDir, filename)

			wavBytes, err := audio.EncodeWAV(perturbedSamples, 16000)
			if err != nil {
				return nil, fmt.Errorf("failed encoding perturbed wav %s: %w", filename, err)
			}
			if err := os.WriteFile(wavPath, wavBytes, 0644); err != nil {
				return nil, fmt.Errorf("failed saving perturbed wav %s: %w", filename, err)
			}

			exp := b.Expected
			if v.Expected != "" {
				exp = v.Expected
			}
			dur := float64(len(perturbedSamples)) / 16000.0
			manifest.Items = append(manifest.Items, EvalItem{
				ID:           fmt.Sprintf("T2_%s_%s", b.BaseID, v.Suffix),
				Description:  fmt.Sprintf("[%s] %s (%s)", v.Description, b.Description, b.Voice),
				Mode:         b.Mode,
				Text:         b.Text,
				AltText:      b.AltText,
				Voice:        b.Voice,
				SpeakingRate: b.SpeakingRate,
				Accent:       b.Accent,
				Category:     b.Category,
				SNR:          v.SNR,
				NoiseType:    v.NoiseType,
				Expected:     exp,
				WavFile:      filename,
				DurationSec:  dur,
			})
		}
	}

	// Step 3: Add sub-gate ambient noise clips (DROPPED_BY_GATE)
	// Sub-gate 1: Ambient room hiss (-48 dBFS)
	hissSamples := audio.GenerateWhiteNoise(16000*2, 777)
	scaleToDBFS(hissSamples, -48)
	hissBytes, _ := audio.EncodeWAV(hissSamples, 16000)
	hissPath := filepath.Join(outDir, "t2_subgate_hiss.wav")
	_ = os.WriteFile(hissPath, hissBytes, 0644)
	manifest.Items = append(manifest.Items, EvalItem{
		ID:          "T2_SUBGATE_HISS",
		Description: "[Sub-Gate] Low-level ambient room hiss (-48 dBFS)",
		Mode:        ModeCommand,
		Text:        "",
		Voice:       "synthetic",
		NoiseType:   audio.NoiseWhite,
		Category:    "noise_gate",
		Expected:    "DROPPED_BY_GATE",
		WavFile:     "t2_subgate_hiss.wav",
		DurationSec: 2.0,
	})

	// Sub-gate 2: Light keyboard clicks, loudest 30 ms at -46 dBFS, below the -42 dBFS gate
	clicksSamples := audio.GenerateTypingNoise(16000*2, 16000, 888)
	scaleToDBFS(clicksSamples, -46)
	clicksBytes, _ := audio.EncodeWAV(clicksSamples, 16000)
	clicksPath := filepath.Join(outDir, "t2_subgate_clicks.wav")
	_ = os.WriteFile(clicksPath, clicksBytes, 0644)
	manifest.Items = append(manifest.Items, EvalItem{
		ID:          "T2_SUBGATE_CLICKS",
		Description: "[Sub-Gate] Distant keyboard typing clicks below gate threshold",
		Mode:        ModeCommand,
		Text:        "",
		Voice:       "synthetic",
		NoiseType:   audio.NoiseTyping,
		Category:    "noise_gate",
		Expected:    "DROPPED_BY_GATE",
		WavFile:     "t2_subgate_clicks.wav",
		DurationSec: 2.0,
	})

	// Save manifest JSON
	manBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed marshaling tier2 manifest: %w", err)
	}
	manPath := filepath.Join(outDir, "manifest.json")
	if err := os.WriteFile(manPath, manBytes, 0644); err != nil {
		return nil, fmt.Errorf("failed saving tier2 manifest JSON: %w", err)
	}

	log.Printf("[Tier2 Generator] Successfully created %d test items in %s", len(manifest.Items), outDir)
	return manifest, nil
}

// GenerateTier1Corpus generates the Tier-1 smoke test corpus and manifest.
func GenerateTier1Corpus(outDir string, runner engine.SpeechEngine) (*EvalManifest, error) {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, err
	}
	exact := 0.0
	cases := []EvalItem{
		{ID: "T1_CMD_SETTINGS", Description: "Command: Open Settings (US Female voice)", Mode: ModeCommand,
			Text: "open settings", Voice: "af_heart", Expected: "NAV_SETTINGS", WavFile: "t1_cmd_settings.wav"},
		{ID: "T1_CMD_START_REC", Description: "Command: Start Recording (US Male voice)", Mode: ModeCommand,
			Text: "start recording", Voice: "am_adam", Expected: "START_RECORDING", WavFile: "t1_cmd_start_recording.wav"},
		{ID: "T1_CMD_NEGATION", Description: "Command: Negation Rejection (US Female voice)", Mode: ModeCommand,
			Text: "do not stop recording", Voice: "af_heart", Expected: "REJECT:NEGATION_DETECTED", WavFile: "t1_cmd_negation.wav"},
		{ID: "T1_DICT_BENCH", Description: "Dictation: Technical phrase (British Female voice)", Mode: ModeDictation,
			Text: "The voice pipeline test bench is operational and ready.", Voice: "bf_emma", Expected: "pipeline test bench",
			MaxErrorRate: &exact, WavFile: "t1_dict_bench.wav"},
	}

	manifest := &EvalManifest{Version: "1.0", Tier: "tier-1-smoke"}
	for _, c := range cases {
		samples, err := LoadOrSynthesizeClip(filepath.Join(outDir, c.WavFile), runner, engine.TTSRequest{
			Text: c.Text, Voice: c.Voice, Speed: 1.0, SampleRate: 24000,
		})
		if err != nil {
			return nil, fmt.Errorf("%s: %w", c.ID, err)
		}
		c.DurationSec = float64(len(samples)) / 16000.0
		manifest.Items = append(manifest.Items, c)
	}

	// Low-level hiss plus faint periodic clicks, both below the gate.
	noise := make([]float32, 16000*2)
	for i := range noise {
		noise[i] = float32((float64(i%997)/997.0 - 0.5) * 0.006)
		if i%3200 == 0 {
			noise[i] += 0.015
		}
	}
	noiseBytes, err := audio.EncodeWAV(noise, 16000)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(outDir, "t1_noise_below_gate.wav"), noiseBytes, 0644); err != nil {
		return nil, err
	}
	manifest.Items = append(manifest.Items, EvalItem{
		ID:          "T1_NOISE_GATE",
		Description: "Acoustic Noise: Low-level typing/room chatter below gate threshold",
		Mode:        ModeCommand,
		Voice:       "synthetic",
		Expected:    "DROPPED_BY_GATE",
		WavFile:     "t1_noise_below_gate.wav",
		DurationSec: 2.0,
	})

	manBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(outDir, "manifest.json"), manBytes, 0644); err != nil {
		return nil, err
	}
	return manifest, nil
}


