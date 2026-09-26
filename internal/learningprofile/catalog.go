package learningprofile

import "strconv"

// Official item bank for website autonomous assessment.
// Stems are copied from the checked Word extracts; do not rewrite.

const (
	ProgramSelfAssessment = "platform_self_assessment"

	InstrumentDL  = "DL-C56-v1"
	InstrumentSDL = "SDL-C20-v1"

	DimCriticalThinking           = "critical_thinking_disposition"
	DimProblemSolving             = "problem_solving_ability"
	DimKnowledgeTransfer          = "knowledge_transfer_ability"
	DimLearningMotivation         = "learning_motivation"
	DimPlanningAndImplementation  = "planning_and_implementation"
	DimSelfManagement             = "self_management"
	DimInterpersonalCommunication = "interpersonal_communication"

	ScoringForward = "forward"

	WavePretest  = "pretest"
	WavePosttest = "posttest"

	StatusDraft     = "draft"
	StatusSubmitted = "submitted"
)

type LikertOption struct {
	Value int    `json:"value"`
	Label string `json:"label"`
}

type InstrumentSpec struct {
	Code      string
	Name      string
	Version   string
	Program   string
	ScaleMin  int
	ScaleMax  int
	ItemCount int
	Options   []LikertOption
}

type ItemSpec struct {
	InstrumentCode     string
	ItemCode           string
	SourceItemNo       string
	Stem               string
	PrimaryDimension   string
	SecondaryDimension string
	ScaleMin           int
	ScaleMax           int
	ScoringDirection   string
	SortOrder          int
}

func DLOptions() []LikertOption {
	return []LikertOption{
		{Value: 1, Label: "非常不符合"},
		{Value: 2, Label: "比较不符合"},
		{Value: 3, Label: "一般"},
		{Value: 4, Label: "比较符合"},
		{Value: 5, Label: "非常符合"},
	}
}

func SDLOptions() []LikertOption {
	return []LikertOption{
		{Value: 1, Label: "非常不同意"},
		{Value: 2, Label: "不同意"},
		{Value: 3, Label: "有点不同意"},
		{Value: 4, Label: "一般 / 不确定"},
		{Value: 5, Label: "有点同意"},
		{Value: 6, Label: "同意"},
		{Value: 7, Label: "非常同意"},
	}
}

func Instruments() []InstrumentSpec {
	return []InstrumentSpec{
		{
			Code: InstrumentDL, Name: "C语言深度学习能力调查问卷", Version: "v1",
			Program: ProgramSelfAssessment, ScaleMin: 1, ScaleMax: 5, ItemCount: 56, Options: DLOptions(),
		},
		{
			Code: InstrumentSDL, Name: "自我导向学习能力量表", Version: "v1",
			Program: ProgramSelfAssessment, ScaleMin: 1, ScaleMax: 7, ItemCount: 20, Options: SDLOptions(),
		},
	}
}

func InstrumentByCode(code string) (InstrumentSpec, bool) {
	for _, inst := range Instruments() {
		if inst.Code == code {
			return inst, true
		}
	}
	return InstrumentSpec{}, false
}

func dl(code, secondary, stem, dim string, sort int) ItemSpec {
	return ItemSpec{
		InstrumentCode: InstrumentDL, ItemCode: code, SourceItemNo: code, Stem: stem,
		PrimaryDimension: dim, SecondaryDimension: secondary,
		ScaleMin: 1, ScaleMax: 5, ScoringDirection: ScoringForward, SortOrder: sort,
	}
}

func sdl(n int, secondary, stem, dim string) ItemSpec {
	code := "SDL" + strconv.Itoa(n)
	return ItemSpec{
		InstrumentCode: InstrumentSDL, ItemCode: code, SourceItemNo: strconv.Itoa(n), Stem: stem,
		PrimaryDimension: dim, SecondaryDimension: secondary,
		ScaleMin: 1, ScaleMax: 7, ScoringDirection: ScoringForward, SortOrder: n,
	}
}

// OfficialItems returns the 76 official Likert items in radar order.
func OfficialItems() []ItemSpec {
	ct := DimCriticalThinking
	ps := DimProblemSolving
	kt := DimKnowledgeTransfer
	items := []ItemSpec{
		dl("CT1", "认知成熟", "我会认真听取他人对我所写C语言程序的意见，即使这些意见和我的想法不同。", ct, 1),
		dl("CT2", "认知成熟", "当出现与我原有判断不一致的新信息（例如调试结果）时，我愿意改变自己对该C语言问题的看法。", ct, 2),
		dl("CT3", "认知成熟", "判断一个C语言问题时，我会尽量依据事实做出判断，而不是凭个人偏见。", ct, 3),
		dl("CT4", "认知成熟", "即使同学的编程思路和我不同，我也能够和他们保持良好的合作关系。", ct, 4),
		dl("CT5", "认知成熟", "我会反思自己固有的编程习惯是否影响了我对某个C语言问题的判断。", ct, 5),
		dl("CT6", "认知成熟", "面对一个C语言问题，我会尝试找出不止一种解法。", ct, 6),
		dl("CT7", "认知成熟", "在做编程方案决策时，我会提出很多问题。", ct, 7),
		dl("CT8", "认知成熟", "我相信大多数C语言问题都存在不止一种解法。", ct, 8),
		dl("CT9", "投入性", "我会主动寻找运用C语言解决实际问题的机会。", ct, 9),
		dl("CT10", "投入性", "我对多种类型的C语言编程问题（如数据结构、算法、文件处理等）感兴趣。", ct, 10),
		dl("CT11", "投入性", "我能够将所学的C语言知识和多种不同类型的问题联系起来。", ct, 11),
		dl("CT12", "投入性", "我喜欢寻找有挑战性的C语言问题的答案。", ct, 12),
		dl("CT13", "投入性", "我认为自己是一个善于解决C语言问题的人。", ct, 13),
		dl("CT14", "投入性", "我有信心针对C语言问题得出合理的结论。", ct, 14),
		dl("CT15", "投入性", "我能够把所学的C语言知识应用到多种不同的任务中。", ct, 15),
		dl("CT16", "投入性", "我能够清楚地解释自己解决C语言问题的思路。", ct, 16),
		dl("CT17", "投入性", "在澄清一个C语言解决方案时，我会提出有价值的问题。", ct, 17),
		dl("CT18", "投入性", "我能够清晰准确地描述一个C语言问题。", ct, 18),
		dl("CT19", "投入性", "面对一个C语言编程任务，我会坚持到底，直到把它解决好。", ct, 19),
		dl("CT20", "创新性", "我喜欢学习C语言中的很多不同知识点。", ct, 20),
		dl("CT21", "创新性", "在学习C语言的过程中，我会提出很多问题。", ct, 21),
		dl("CT22", "创新性", "我认为持续掌握最新的编程知识很重要。", ct, 22),
		dl("CT23", "创新性", "我喜欢解决C语言编程问题。", ct, 23),
		dl("CT24", "创新性", "即使不是在上课，我也喜欢学习C语言相关的知识。", ct, 24),
		dl("CT25", "创新性", "即使结果可能不理想，我也愿意弄清楚一个C语言问题背后的真正原因。", ct, 25),
		dl("CT26", "创新性", "为了找到一个C语言问题的正确答案，我愿意额外花费时间和精力。", ct, 26),
		dl("PS1", "问题界定", "面对C语言编程任务时，我认为自己能够明确需要解决的核心问题。", ps, 27),
		dl("PS2", "问题界定", "面对C语言编程任务时，我认为自己能够确定编程任务的输入信息。", ps, 28),
		dl("PS3", "问题界定", "面对C语言编程任务时，我认为自己能够确定程序需要产生的预期结果。", ps, 29),
		dl("PS4", "策略识别", "面对C语言编程问题时，我认为自己能够识别一种可能的解决途径。", ps, 30),
		dl("PS5", "策略识别", "当常用的解决途径不适用时，我认为自己能够识别其他可能的解决途径。", ps, 31),
		dl("PS6", "策略识别", "面对同一个C语言编程问题时，我认为自己能够识别不同的解决途径。", ps, 32),
		dl("PS7", "方案提出", "我认为自己能够针对C语言编程问题提出具体的解决方案。", ps, 33),
		dl("PS8", "方案提出", "我认为自己能够将解决思路组织为有序的实现步骤。", ps, 34),
		dl("PS9", "方案提出", "我认为自己能够根据任务要求确定解决方案所需的程序结构。", ps, 35),
		dl("PS10", "方案评价", "我认为自己能够判断一个解决方案的逻辑是否合理。", ps, 36),
		dl("PS11", "方案评价", "我认为自己能够判断一个解决方案是否可以在当前条件下实施。", ps, 37),
		dl("PS12", "方案评价", "我认为自己能够识别一个解决方案不适用的情形。", ps, 38),
		dl("PS13", "方案实施", "我认为自己能够将解决方案中的步骤编写为相应的C语言代码。", ps, 39),
		dl("PS14", "方案实施", "我认为自己能够将分别编写的代码部分组合成一个完整的程序。", ps, 40),
		dl("PS15", "方案实施", "当代码实现偏离原定方案时，我认为自己能够调整代码。", ps, 41),
		dl("PS16", "结果评价", "程序运行后，我认为自己能够判断输出结果是否符合任务目标。", ps, 42),
		dl("PS17", "结果评价", "我认为自己能够识别程序输出结果与预期结果之间的差异。", ps, 43),
		dl("PS18", "结果评价", "根据程序运行结果，我认为自己能够判断解决方案是否需要修改。", ps, 44),
		dl("KT1", "近迁移", "面对与课堂例题相近的编程任务，我认为自己能够运用已经学过的C语言知识。", kt, 45),
		dl("KT2", "近迁移", "面对与课堂示例有所不同的同类编程任务，我认为自己能够运用学过的方法完成。", kt, 46),
		dl("KT3", "近迁移", "面对解题结构相似的编程任务，我认为自己能够采用以前学过的解题方法。", kt, 47),
		dl("KT4", "适应性迁移", "面对新的编程任务，我认为自己能够识别它与以往任务之间的关键差异。", kt, 48),
		dl("KT5", "适应性迁移", "面对新的编程任务，我认为自己能够从已经学过的方法中选择可供借鉴的方法。", kt, 49),
		dl("KT6", "适应性迁移", "我认为自己能够根据新任务的要求调整所选择的方法。", kt, 50),
		dl("KT7", "整合性迁移", "学习新的C语言内容时，我认为自己能够理解它与已学内容之间的联系。", kt, 51),
		dl("KT8", "整合性迁移", "面对综合性编程任务时，我认为自己能够判断需要运用哪些已经学过的知识。", kt, 52),
		dl("KT9", "整合性迁移", "我认为自己能够在同一个程序中综合运用不同课程单元的知识。", kt, 53),
		dl("KT10", "反思性迁移", "完成编程任务后，我认为自己能够分析所采用的方法是否有效。", kt, 54),
		dl("KT11", "反思性迁移", "完成编程任务后，我认为自己能够总结其中值得借鉴的做法。", kt, 55),
		dl("KT12", "反思性迁移", "面对新的编程任务时，我认为自己能够运用以往任务中获得的经验。", kt, 56),
	}

	mot := DimLearningMotivation
	plan := DimPlanningAndImplementation
	sm := DimSelfManagement
	ic := DimInterpersonalCommunication
	items = append(items,
		sdl(1, "学习动机", "我知道自己在 C 语言课程中需要学习什么。", mot),
		sdl(2, "学习动机", "无论学习结果如何，我仍然愿意继续学习 C 语言。", mot),
		sdl(3, "学习动机", "我希望自己在 C 语言学习中不断进步，并取得更好的表现。", mot),
		sdl(4, "学习动机", "我在 C 语言学习中的成功和失败都会激励我继续学习。", mot),
		sdl(5, "学习动机", "我喜欢主动寻找 C 语言学习问题的答案。", mot),
		sdl(6, "学习动机", "即使在 C 语言学习中遇到困难，我也不会轻易放弃。", mot),
		sdl(7, "计划与执行", "我能够主动设定自己的 C 语言学习目标。", plan),
		sdl(8, "计划与执行", "我知道哪些学习策略适合自己实现 C 语言学习目标。", plan),
		sdl(9, "计划与执行", "我会根据重要性和难度安排 C 语言学习任务的优先顺序。", plan),
		sdl(10, "计划与执行", "无论在课堂学习、上机实践还是课后自主学习中，我都能够按照自己的学习计划进行学习。", plan),
		sdl(11, "计划与执行", "我善于安排和控制自己的 C 语言学习时间。", plan),
		sdl(12, "计划与执行", "我知道如何寻找适合 C 语言学习的资源。", plan),
		sdl(13, "自我管理", "我能够将新学到的 C 语言知识与已有知识或实际经验联系起来。", sm),
		sdl(14, "自我管理", "我了解自己在 C 语言学习中的优势和不足。", sm),
		sdl(15, "自我管理", "我能够监控自己的 C 语言学习进度。", sm),
		sdl(16, "自我管理", "我能够评价自己的 C 语言学习效果。", sm),
		sdl(17, "人际沟通", "与教师或同学的交流能够帮助我规划后续 C 语言学习。", ic),
		sdl(18, "人际沟通", "我愿意通过与教师和同学交流，理解不同的编程思路和学习方法。", ic),
		sdl(19, "人际沟通", "我能够在讨论或汇报中清楚表达自己的编程思路。", ic),
		sdl(20, "人际沟通", "我能够通过文字、代码注释或学习记录清楚表达自己的学习过程和问题。", ic),
	)
	return items
}

func ItemsByInstrument(code string) []ItemSpec {
	var out []ItemSpec
	for _, item := range OfficialItems() {
		if item.InstrumentCode == code {
			out = append(out, item)
		}
	}
	return out
}

func ItemByCode(code string) (ItemSpec, bool) {
	for _, item := range OfficialItems() {
		if item.ItemCode == code {
			return item, true
		}
	}
	return ItemSpec{}, false
}

func DimensionOrder() []string {
	return []string{
		DimCriticalThinking,
		DimProblemSolving,
		DimKnowledgeTransfer,
		DimLearningMotivation,
		DimPlanningAndImplementation,
		DimSelfManagement,
		DimInterpersonalCommunication,
	}
}

// RadarDimensionSpec is one fixed radar axis: official code, instrument, item count, and raw scale.
type RadarDimensionSpec struct {
	Code           string
	InstrumentCode string
	ItemCount      int
	ScaleMin       int
	ScaleMax       int
}

func RadarDimensionSpecs() []RadarDimensionSpec {
	order := DimensionOrder()
	out := make([]RadarDimensionSpec, 0, len(order))
	for _, code := range order {
		var inst string
		count := 0
		scaleMin, scaleMax := 0, 0
		for _, item := range OfficialItems() {
			if item.PrimaryDimension != code {
				continue
			}
			if count == 0 {
				inst = item.InstrumentCode
				scaleMin = item.ScaleMin
				scaleMax = item.ScaleMax
			}
			count++
		}
		if count == 0 || inst == "" {
			continue
		}
		out = append(out, RadarDimensionSpec{
			Code:           code,
			InstrumentCode: inst,
			ItemCount:      count,
			ScaleMin:       scaleMin,
			ScaleMax:       scaleMax,
		})
	}
	return out
}
