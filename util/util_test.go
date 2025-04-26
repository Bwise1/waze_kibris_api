package util

import (
	"testing"
	"time"
)

func TestPolyLineDecoder(t *testing.T) {
	encoded := "qlvcbAwspp~@}AxAwKfKcUhUoYbVq]|X{UtQgc@zZ_KrGoFjCwDrBsCpDw@fAuAxBcBpGuBlCgB|@qCPcCYQeDGmDR_Dh@gLBoKeAuPqCca@kEs`@kDcRkB}JkB}JqAoGa@oB}AyHyEmU}Pov@qLsj@aCwLoFoYoNku@sCwJ{A}FuIgMqIwHsFqA_FsCuEqF{CkHsAuIIaJpAeKhDuIpAyMFiMa@uJwA{JyFiXiCkLuEgQmOiq@c[wvAya@okBaDcO_Kae@o@wCaHub@aCoUiAiTa@yl@t@ol@jBce@rDua@lBqP~ByQzDe]rg@clEjDg_@nAuMd@eF~@sLlDuXrIgp@|UcpB`Jiy@~Dq`@`B{O|Iqu@jd@i~DnGsj@pIyt@vBmTdBsOnC}YrBaZpBaZl@yIh@sTLkJHcL?kB?eBYmb@EoCc@{TG{Dw@ie@cAiYk@cOgHgqAQkEyJ{fB_NmdCs@wMsAaQiDgt@i@iLqKksBeGsiAaEes@WkEO{CcDwm@WkFwDqo@s@gMkDgo@gT_~DsBk_@a@uHqC{g@]gGyHsmAqKyrAcAcMoVywCo@yHW{Cu@gJo@wMs@eOGwAuAyVwNovBs@wJuLgbBoAmO}Eom@IgA}BmZmB{VyAkRs@cJwJmoAma@gkFwW_iDcGwv@eH{}@sI{gAmRqdC}R_lCQeDiCqf@{AiUcBqNiCeHcDqFkCyAsBaCaB_Eq@wEBaFx@uEtA_DjAoHXuJCkTuB}Ww`@okFwg@_rGoSimC_@_F{AoRgCu\\_@{EkAwSqAkTEyNGaMVoN\\yFjEot@rJw_BBc@fIkjAhCcXl@aGl@mGrA}RTyH@}I_@iL]}@aKaAuCI}AJ}ElA}GhCmKzEwHpCaFvDyFhEcJjLeD`GoBjDeDzHyAhDaChIw@`DeE`QoEvMwDdI_H|JkJnL_MdLiRxRyq@ns@qZj[eTnWeDlDqNhOqDzDyInI"
	result, err := DecodePolyLines(encoded)
	if err != nil {
		t.Fatalf("Decoding returned error %v", err)
	}
	t.Logf("decode %v", result)
}

func TestValhallaDecode(t *testing.T){
	encoded:="qlvcbAwspp~@}AxAwKfKcUhUoYbVq]|X{UtQgc@zZ_KrGoFjCwDrBsCpDw@fAuAxBcBpGuBlCgB|@qCPcCYQeDGmDR_Dh@gLBoKeAuPqCca@kEs`@kDcRkB}JkB}JqAoGa@oB}AyHyEmU}Pov@qLsj@aCwLoFoYoNku@sCwJ{A}FuIgMqIwHsFqA_FsCuEqF{CkHsAuIIaJpAeKhDuIpAyMFiMa@uJwA{JyFiXiCkLuEgQmOiq@c[wvAya@okBaDcO_Kae@o@wCaHub@aCoUiAiTa@yl@t@ol@jBce@rDua@lBqPUqAa@mBcAkCy@mAkAiAaBeAuD}A}Qw@gSiAuT_A_ESmFHsJrBaJlEeT|Nwi@~]_uAd|@uO`Kow@tg@ajA~w@cOtLg[z\\oF~Fqc@tl@aUjWiUxO_h@~[uMpHoI|HgA~Ao@x@_Cr@eCG{BgAmTq@}WX}Yd@}G@k_@f@}Wd@as@lAgUvCmFl@yUnFqRlGeQzFaKnAeBTeRVkh@jBoCNyCVqBKoj@bBcEPoDEqDbA}EpAaFjD}j@vn@uK~I{L~G_NxD}YlIaRbE_Fv@qDBiH_@qHeAw|@cT}KSeFUoMcAqM]wHGoEA{[c@e_@Ter@NqRCcNIoGAwWA_KAsGpBiDlAeEvCwBvCiAnFi@dHZ|GlBxGbDzDtFbDzFXbG_AvFmEhCkGrA{HrAyH{@ih@M_MRkc@NyKNmLl@_YpDep@zBwRrDiYtE}X~Hsc@hKui@rj@}eCjXmuAhn@{xC`u@meDfRmu@tL}c@bLma@fTmv@na@qyAnWo}@rHmY|Gy[lMiu@dJsi@tFm[dIwd@jHq`@hJcd@rOkj@|Ske@t[sk@|o@goAfc@wx@x[am@jUad@hJ}RtDsMjCwL`CoLfBwLpAyK~@_Lh@mIX{KJsKGeHC_C]_Jo@aLeA_M_DiZoG{m@}CwM}EmKsCoBuB{CoAeE]yERyEdAkE`@{@l@yIf@eJ@uJ}@gSiB{PmNqvAcHou@wD_k@kKwtBiGqiAyC}x@aCsj@}GcvAyFsaAwGa}@YuD}Ekl@qAwOsCe[uBgMcDaMcHiPwC_D}AwDq@qEAwEl@qEx@_CnBeWTkMoA_s@s@uj@e@uj@Vwv@b@gb@bAwYzHmnAjEmo@fFqs@fDih@fGqp@vEgr@jAeQfGkz@tD}s@XwaAQq]aB{[yEcn@oJmfAgKaq@kKcj@kAgGqr@wvD_M_w@wDa\\Q_BsAgLoCol@cAcq@i@uK}C{MgDsJiCgDcBiEaAiGEsGv@kGtAcEvBgDzGcQrBoNn@eOAaJQe}@c@a~@G_Q_@sNy@{PiAiOuAsKu@kHK}@UuBuHmd@cBwIs@oDeK_[{EqJsCoCsCqCuJmEoCc@yDqBuCqD_B_Fc@{FHwCfA}FfCqEvDqChEaAnENzG_ElJcEnVuShUaPlQkP`IsIpqAcvAbRgS|wBa}Blh@ej@r^_`@jIyIx[y\\nJoLrZ_ZtXcZxCaDbGkHfV{YvOuPhDqDrSaS"
	result, err := DecodeValhallaPolyline6(encoded)
	if err != nil {
		t.Fatalf("Decoding returned error %v", err)
	}
	t.Logf("decode %v", result)
}
func TestFormatTime(t *testing.T) {
	testTime := time.Date(2025, 4, 5, 14, 30, 45, 0, time.UTC)

	// Test cases with different formats
	testCases := []struct {
		name           string
		format         string
		expectedResult string
	}{
		{"RFC3339", time.RFC3339, "2025-04-05T14:30:45Z"},
		{"Simple Date", "2006-01-02", "2025-04-05"},
		{"Time Only", "15:04:05", "14:30:45"},
		{"Date and Time", "2006-01-02 15:04:05", "2025-04-05 14:30:45"},
		{"Custom Format", "Mon Jan 2 15:04:05 MST 2006", "Sat Apr 5 14:30:45 UTC 2025"},
		{"Short Date", "Jan 2", "Apr 5"},
		{"Day of Week", "Monday", "Saturday"},
		{"Month and Year", "January 2006", "April 2025"},
		{"Kitchen Time", time.Kitchen, "2:30PM"},
		{"Year Only", "2006", "2025"},
		{"Month Only", "January", "April"},
		{"Day Only", "2", "5"},
		{"Unix Timestamp", time.UnixDate, "Sat Apr  5 14:30:45 UTC 2025"},
		{"RFC1123", time.RFC1123, "Sat, 05 Apr 2025 14:30:45 UTC"},
		{"ISO8601", "2006-01-02T15:04:05-0700", "2025-04-05T14:30:45+0000"},
		{"Empty Format", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatTime(tc.format, testTime)

			if result != tc.expectedResult {
				t.Errorf("formatTime(%q, %v) = %q; want %q",
					tc.format, testTime, result, tc.expectedResult)
			}
		})
	}

}
