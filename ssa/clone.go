package ssa

import (
	"fmt"
	"go/types"
)

type cloner struct {
	// OPT(dh): benchmark the effect of using a single
	// map[unsafe.Pointer]unsafe.Pointer
	nprog  *Program
	pkgMap map[*Package]*Package

	// XXX rename
	theMap map[interface{}]interface{}

	bbMap map[*BasicBlock]*BasicBlock
}

func (c *cloner) remember(old, new interface{}) {
	c.theMap[old] = new
}

func (c *cloner) clonePkg(pkg *Package) *Package {
	// XXX is unsafe special?
	if pkg == nil {
		return nil
	}

	if npkg, ok := c.pkgMap[pkg]; ok {
		return npkg
	}

	npkg := &Package{
		Prog:    c.nprog,
		Pkg:     pkg.Pkg,
		Members: map[string]Member{},
		values:  map[types.Object]Value{},
		debug:   pkg.debug,
	}
	c.pkgMap[pkg] = npkg
	npkg.init = c.cloneFunction(pkg.init)

	for key, m := range pkg.Members {
		npkg.Members[key] = c.cloneMember(m)
	}
	for obj, v := range pkg.values {
		npkg.values[obj] = c.cloneValue(v)
	}
	return npkg
}

func (c *cloner) cloneFunction(fn *Function) *Function {
	// called directly
	if fn == nil {
		return nil
	}
	if nfn, ok := c.theMap[fn]; ok {
		return nfn.(*Function)
	}

	nfn := &Function{
		name:      fn.name,
		object:    fn.object,
		method:    fn.method,
		Signature: fn.Signature,
		pos:       fn.pos,

		Synthetic: fn.Synthetic,
		syntax:    fn.syntax,
		Pkg:       c.clonePkg(fn.Pkg),
		Prog:      c.nprog,
	}
	c.remember(fn, nfn)
	nfn.parent = c.cloneFunction(fn.parent)
	if fn.AnonFuncs != nil {
		nfn.AnonFuncs = make([]*Function, len(fn.AnonFuncs))
		for i, anon := range fn.AnonFuncs {
			nfn.AnonFuncs[i] = c.cloneFunction(anon)
		}
	}

	if fn.Params != nil {
		nfn.Params = make([]*Parameter, len(fn.Params))
		for i, x := range fn.Params {
			nfn.Params[i] = c.cloneParameter(x)
		}
	}
	if fn.FreeVars != nil {
		nfn.FreeVars = make([]*FreeVar, len(fn.FreeVars))
		for i, x := range fn.FreeVars {
			nfn.FreeVars[i] = c.cloneFreeVar(x)
		}
	}
	if fn.Locals != nil {
		nfn.Locals = make([]*Alloc, len(fn.Locals))
		for i, x := range fn.Locals {
			nfn.Locals[i] = c.cloneAlloc(x)
		}
	}
	if fn.Blocks != nil {
		nfn.Blocks = make([]*BasicBlock, len(fn.Blocks))
		for i, x := range fn.Blocks {
			nfn.Blocks[i] = c.cloneBasicBlock(x)
		}
	}
	nfn.referrers = c.cloneInstructions(fn.referrers)
	if fn.namedResults != nil {
		nfn.namedResults = make([]*Alloc, len(fn.namedResults))
		for i, x := range fn.namedResults {
			nfn.namedResults[i] = c.cloneAlloc(x)
		}
	}

	nfn.Recover = c.cloneBasicBlock(fn.Recover)

	return nfn
}

func (c *cloner) cloneMember(m Member) Member {
	switch m := m.(type) {
	case *NamedConst:
		return c.cloneNamedConst(m)
	case *Global:
		return c.cloneGlobal(m)
	case *Function:
		return c.cloneFunction(m)
	case *Type:
		return c.cloneType(m)
	default:
		panic(fmt.Sprintf("internal error: unexpected type %T", m))
	}
}

func (c *cloner) cloneNamedConst(m *NamedConst) *NamedConst {
	// called directly
	if m == nil {
		return nil
	}
	if nm, ok := c.theMap[m]; ok {
		return nm.(*NamedConst)
	}

	nm := &NamedConst{
		object: m.object,
	}
	c.remember(m, nm)
	nm.Value = c.cloneConst(m.Value)
	nm.pkg = c.clonePkg(m.pkg)
	return nm
}

func (c *cloner) cloneType(m *Type) *Type {
	// called directly
	if m == nil {
		return nil
	}
	if nm, ok := c.theMap[m]; ok {
		return nm.(*Type)
	}

	nm := &Type{
		object: m.object,
	}
	c.remember(m, nm)
	nm.pkg = c.clonePkg(m.pkg)
	return nm
}

func (c *cloner) cloneValue(val Value) Value {
	if val == nil {
		return nil
	}
	if nval, ok := c.theMap[val]; ok {
		return nval.(Value)
	}

	switch val := val.(type) {
	case *Function:
		return c.cloneFunction(val)
	case *FreeVar:
		return c.cloneFreeVar(val)
	case *Parameter:
		return c.cloneParameter(val)
	case *Const:
		return c.cloneConst(val)
	case *Global:
		return c.cloneGlobal(val)
	case *Builtin:
		return c.cloneBuiltin(val)
	case *Alloc:
		return c.cloneAlloc(val)
	case *Sigma:
		return c.cloneSigma(val)
	case *Phi:
		return c.clonePhi(val)
	case *Call:
		return c.cloneCall(val)
	case *BinOp:
		return c.cloneBinOp(val)
	case *UnOp:
		return c.cloneUnOp(val)
	case *ChangeType:
		return c.cloneChangeType(val)
	case *ChangeInterface:
		return c.cloneChangeInterface(val)
	case *MakeInterface:
		return c.cloneMakeInterface(val)
	case *MakeClosure:
		return c.cloneMakeClosure(val)
	case *MakeMap:
		return c.cloneMakeMap(val)
	case *MakeChan:
		return c.cloneMakeChan(val)
	case *MakeSlice:
		return c.cloneMakeSlice(val)
	case *Slice:
		return c.cloneSlice(val)
	case *FieldAddr:
		return c.cloneFieldAddr(val)
	case *Field:
		return c.cloneField(val)
	case *IndexAddr:
		return c.cloneIndexAddr(val)
	case *Index:
		return c.cloneIndex(val)
	case *Lookup:
		return c.cloneLookup(val)
	case *Select:
		return c.cloneSelect(val)
	case *Range:
		return c.cloneRange(val)
	case *Next:
		return c.cloneNext(val)
	case *TypeAssert:
		return c.cloneTypeAssert(val)
	case *Extract:
		return c.cloneExtract(val)
	case *Convert:
		return c.cloneConvert(val)
	case nil:
		return nil
	default:
		panic(fmt.Sprintf("internal error: unexpected type %T", val))
	}
}

func (c *cloner) cloneFreeVar(val *FreeVar) *FreeVar {
	// called directly
	if val == nil {
		return nil
	}
	if nval, ok := c.theMap[val]; ok {
		return nval.(*FreeVar)
	}

	nval := &FreeVar{
		name: val.name,
		typ:  val.typ,
		pos:  val.pos,
	}
	c.remember(val, nval)
	nval.parent = c.cloneFunction(val.parent)
	val.referrers = c.cloneInstructions(val.referrers)
	return nval
}

func (c *cloner) cloneParameter(val *Parameter) *Parameter {
	// called directly
	if val == nil {
		return nil
	}
	if nval, ok := c.theMap[val]; ok {
		return nval.(*Parameter)
	}

	nval := &Parameter{
		name:   val.name,
		object: val.object,
		typ:    val.typ,
		pos:    val.pos,
	}
	c.remember(val, nval)
	nval.parent = c.cloneFunction(val.parent)
	nval.referrers = c.cloneInstructions(val.referrers)
	return nval
}

func (c *cloner) cloneConst(val *Const) *Const { return val }

func (c *cloner) cloneGlobal(val *Global) *Global {
	if val == nil {
		return nil
	}
	if nval, ok := c.theMap[val]; ok {
		return nval.(*Global)
	}

	nval := &Global{
		name:   val.name,
		object: val.object,
		typ:    val.typ,
		pos:    val.pos,
	}
	c.remember(val, nval)
	nval.Pkg = c.clonePkg(val.Pkg)
	return nval
}

func (c *cloner) cloneBuiltin(val *Builtin) *Builtin { return val }

func (c *cloner) cloneAlloc(val *Alloc) *Alloc {
	// called directly
	if val == nil {
		return nil
	}
	if nval, ok := c.theMap[val]; ok {
		return nval.(*Alloc)
	}

	nval := &Alloc{
		Comment: val.Comment,
		Heap:    val.Heap,
		index:   val.index,
	}
	c.remember(val, nval)
	nval.register = c.cloneRegister(val.register)
	return nval
}

func (c *cloner) cloneSigma(val *Sigma) *Sigma {
	nval := &Sigma{
		Branch: val.Branch,
	}
	c.remember(val, nval)
	nval.register = c.cloneRegister(val.register)
	nval.X = c.cloneValue(val.X)
	return nval
}

func (c *cloner) clonePhi(val *Phi) *Phi {
	nval := &Phi{
		Comment: val.Comment,
	}
	c.remember(val, nval)
	nval.register = c.cloneRegister(val.register)
	nval.Edges = c.cloneValues(val.Edges)
	return nval
}

func (c *cloner) cloneCall(val *Call) *Call {
	nval := &Call{}
	c.remember(val, nval)
	nval.register = c.cloneRegister(val.register)
	nval.Call = c.cloneCallCommon(val.Call)
	return nval
}

func (c *cloner) cloneCallCommon(call CallCommon) CallCommon {
	ncall := CallCommon{
		Method: call.Method,
		pos:    call.pos,
	}
	ncall.Value = c.cloneValue(call.Value)
	ncall.Args = c.cloneValues(call.Args)
	return ncall
}

func (c *cloner) cloneBinOp(val *BinOp) *BinOp {
	nval := &BinOp{
		Op: val.Op,
	}
	c.remember(val, nval)
	nval.register = c.cloneRegister(val.register)
	nval.X = c.cloneValue(val.X)
	nval.Y = c.cloneValue(val.Y)
	return nval
}

func (c *cloner) cloneUnOp(val *UnOp) *UnOp {
	nval := &UnOp{
		Op:      val.Op,
		CommaOk: val.CommaOk,
	}
	c.remember(val, nval)
	nval.register = c.cloneRegister(val.register)
	nval.X = c.cloneValue(val.X)
	return nval
}

func (c *cloner) cloneChangeType(val *ChangeType) *ChangeType {
	nval := &ChangeType{}
	c.remember(val, nval)
	nval.register = c.cloneRegister(val.register)
	nval.X = c.cloneValue(val.X)
	return nval
}

func (c *cloner) cloneChangeInterface(val *ChangeInterface) *ChangeInterface {
	nval := &ChangeInterface{}
	c.remember(val, nval)
	nval.register = c.cloneRegister(val.register)
	nval.X = c.cloneValue(val.X)
	return nval
}

func (c *cloner) cloneMakeInterface(val *MakeInterface) *MakeInterface {
	nval := &MakeInterface{}
	c.remember(val, nval)
	nval.register = c.cloneRegister(val.register)
	nval.X = c.cloneValue(val.X)
	return nval
}

func (c *cloner) cloneMakeClosure(val *MakeClosure) *MakeClosure {
	nval := &MakeClosure{}
	c.remember(val, nval)
	nval.register = c.cloneRegister(val.register)
	nval.Fn = c.cloneValue(val.Fn)
	nval.Bindings = c.cloneValues(val.Bindings)
	return nval
}

func (c *cloner) cloneMakeMap(val *MakeMap) *MakeMap {
	nval := &MakeMap{}
	c.remember(val, nval)
	nval.register = c.cloneRegister(val.register)
	nval.Reserve = c.cloneValue(val.Reserve)
	return nval
}

func (c *cloner) cloneMakeChan(val *MakeChan) *MakeChan {
	nval := &MakeChan{}
	c.remember(val, nval)
	nval.register = c.cloneRegister(val.register)
	nval.Size = c.cloneValue(val.Size)
	return nval
}

func (c *cloner) cloneMakeSlice(val *MakeSlice) *MakeSlice {
	nval := &MakeSlice{}
	c.remember(val, nval)
	nval.register = c.cloneRegister(val.register)
	nval.Len = c.cloneValue(val.Len)
	nval.Cap = c.cloneValue(val.Cap)
	return nval
}

func (c *cloner) cloneSlice(val *Slice) *Slice {
	nval := &Slice{}
	c.remember(val, nval)
	nval.register = c.cloneRegister(val.register)
	nval.X = c.cloneValue(val.X)
	nval.Low = c.cloneValue(val.Low)
	nval.High = c.cloneValue(val.High)
	nval.Max = c.cloneValue(val.Max)
	return nval
}

func (c *cloner) cloneFieldAddr(val *FieldAddr) *FieldAddr {
	nval := &FieldAddr{
		Field: val.Field,
	}
	c.remember(val, nval)
	nval.register = c.cloneRegister(val.register)
	nval.X = c.cloneValue(val.X)
	return nval
}

func (c *cloner) cloneField(val *Field) *Field {
	nval := &Field{
		Field: val.Field,
	}
	c.remember(val, nval)
	nval.register = c.cloneRegister(val.register)
	nval.X = c.cloneValue(val.X)
	return nval
}

func (c *cloner) cloneIndexAddr(val *IndexAddr) *IndexAddr {
	nval := &IndexAddr{}
	c.remember(val, nval)
	nval.register = c.cloneRegister(val.register)
	nval.X = c.cloneValue(val.X)
	nval.Index = c.cloneValue(val.Index)
	return nval
}

func (c *cloner) cloneIndex(val *Index) *Index {
	nval := &Index{}
	c.remember(val, nval)
	nval.register = c.cloneRegister(val.register)
	nval.X = c.cloneValue(val.X)
	nval.Index = c.cloneValue(val.Index)
	return nval
}

func (c *cloner) cloneLookup(val *Lookup) *Lookup {
	nval := &Lookup{
		CommaOk: val.CommaOk,
	}
	c.remember(val, nval)
	nval.register = c.cloneRegister(val.register)
	nval.X = c.cloneValue(val.X)
	nval.Index = c.cloneValue(val.Index)
	return nval
}

func (c *cloner) cloneSelect(val *Select) *Select {
	nval := &Select{
		Blocking: val.Blocking,
	}
	c.remember(val, nval)
	nval.register = c.cloneRegister(val.register)
	if val.States != nil {
		nval.States = make([]*SelectState, len(val.States))
		for i, state := range val.States {
			nval.States[i] = c.cloneSelectState(state)
		}
	}
	return nval
}

func (c *cloner) cloneSelectState(state *SelectState) *SelectState {
	// called directly
	if state == nil {
		return nil
	}

	nstate := &SelectState{
		Dir:       state.Dir,
		Pos:       state.Pos,
		DebugNode: state.DebugNode,
	}
	nstate.Chan = c.cloneValue(state.Chan)
	nstate.Send = c.cloneValue(state.Send)
	return nstate
}

func (c *cloner) cloneRange(val *Range) *Range {
	nval := &Range{}
	c.remember(val, nval)
	nval.register = c.cloneRegister(val.register)
	nval.X = c.cloneValue(val.X)
	return nval
}

func (c *cloner) cloneNext(val *Next) *Next {
	nval := &Next{
		IsString: val.IsString,
	}
	c.remember(val, nval)
	nval.register = c.cloneRegister(val.register)
	nval.Iter = c.cloneValue(val.Iter)
	return nval
}

func (c *cloner) cloneTypeAssert(val *TypeAssert) *TypeAssert {
	nval := &TypeAssert{
		AssertedType: val.AssertedType,
		CommaOk:      val.CommaOk,
	}
	c.remember(val, nval)
	nval.register = c.cloneRegister(val.register)
	nval.X = c.cloneValue(val.X)
	return nval
}

func (c *cloner) cloneExtract(val *Extract) *Extract {
	nval := &Extract{
		Index: val.Index,
	}
	c.remember(val, nval)
	nval.register = c.cloneRegister(val.register)
	nval.Tuple = c.cloneValue(val.Tuple)
	return nval
}

func (c *cloner) cloneInstruction(instr Instruction) Instruction {
	if instr == nil {
		return nil
	}
	if ninstr, ok := c.theMap[instr]; ok {
		return ninstr.(Instruction)
	}

	switch instr := instr.(type) {
	case *Alloc:
		return c.cloneAlloc(instr)
	case *Sigma:
		return c.cloneSigma(instr)
	case *Phi:
		return c.clonePhi(instr)
	case *Call:
		return c.cloneCall(instr)
	case *BinOp:
		return c.cloneBinOp(instr)
	case *UnOp:
		return c.cloneUnOp(instr)
	case *ChangeType:
		return c.cloneChangeType(instr)
	case *ChangeInterface:
		return c.cloneChangeInterface(instr)
	case *MakeInterface:
		return c.cloneMakeInterface(instr)
	case *MakeClosure:
		return c.cloneMakeClosure(instr)
	case *MakeMap:
		return c.cloneMakeMap(instr)
	case *MakeChan:
		return c.cloneMakeChan(instr)
	case *MakeSlice:
		return c.cloneMakeSlice(instr)
	case *Slice:
		return c.cloneSlice(instr)
	case *FieldAddr:
		return c.cloneFieldAddr(instr)
	case *Field:
		return c.cloneField(instr)
	case *IndexAddr:
		return c.cloneIndexAddr(instr)
	case *Index:
		return c.cloneIndex(instr)
	case *Lookup:
		return c.cloneLookup(instr)
	case *Select:
		return c.cloneSelect(instr)
	case *Range:
		return c.cloneRange(instr)
	case *Next:
		return c.cloneNext(instr)
	case *TypeAssert:
		return c.cloneTypeAssert(instr)
	case *Extract:
		return c.cloneExtract(instr)
	case *Jump:
		return c.cloneJump(instr)
	case *If:
		return c.cloneIf(instr)
	case *Return:
		return c.cloneReturn(instr)
	case *RunDefers:
		return c.cloneRunDefers(instr)
	case *Panic:
		return c.clonePanic(instr)
	case *Go:
		return c.cloneGo(instr)
	case *Defer:
		return c.cloneDefer(instr)
	case *Send:
		return c.cloneSend(instr)
	case *Store:
		return c.cloneStore(instr)
	case *BlankStore:
		return c.cloneBlankStore(instr)
	case *MapUpdate:
		return c.cloneMapUpdate(instr)
	case *DebugRef:
		return c.cloneDebugRef(instr)
	case *Convert:
		return c.cloneConvert(instr)
	case nil:
		return nil
	default:
		panic(fmt.Sprintf("internal error: unexpected type %T", instr))
	}
}

func (c *cloner) cloneConvert(instr *Convert) *Convert {
	ninstr := &Convert{}
	c.remember(instr, ninstr)
	ninstr.register = c.cloneRegister(instr.register)
	ninstr.X = c.cloneValue(instr.X)
	return ninstr
}

func (c *cloner) cloneJump(instr *Jump) *Jump {
	ninstr := &Jump{}
	c.remember(instr, ninstr)
	ninstr.anInstruction = c.cloneAnInstruction(instr.anInstruction)
	return ninstr
}

func (c *cloner) cloneIf(instr *If) *If {
	ninstr := &If{}
	c.remember(instr, ninstr)
	ninstr.anInstruction = c.cloneAnInstruction(instr.anInstruction)
	ninstr.Cond = c.cloneValue(ninstr.Cond)
	return ninstr
}

func (c *cloner) cloneReturn(instr *Return) *Return {
	ninstr := &Return{
		pos: instr.pos,
	}
	c.remember(instr, ninstr)
	ninstr.anInstruction = c.cloneAnInstruction(instr.anInstruction)
	if instr.Results != nil {
		ninstr.Results = make([]Value, len(instr.Results))
		for i, v := range instr.Results {
			ninstr.Results[i] = c.cloneValue(v)
		}
	}
	return ninstr
}

func (c *cloner) cloneRunDefers(instr *RunDefers) *RunDefers {
	ninstr := &RunDefers{}
	c.remember(instr, ninstr)
	ninstr.anInstruction = c.cloneAnInstruction(instr.anInstruction)
	return ninstr
}

func (c *cloner) clonePanic(instr *Panic) *Panic {
	ninstr := &Panic{
		pos: instr.pos,
	}
	c.remember(instr, ninstr)
	ninstr.anInstruction = c.cloneAnInstruction(instr.anInstruction)
	ninstr.X = c.cloneValue(instr.X)
	return ninstr
}

func (c *cloner) cloneGo(instr *Go) *Go {
	ninstr := &Go{
		pos: instr.pos,
	}
	c.remember(instr, ninstr)
	ninstr.anInstruction = c.cloneAnInstruction(instr.anInstruction)
	ninstr.Call = c.cloneCallCommon(instr.Call)
	return ninstr
}

func (c *cloner) cloneDefer(instr *Defer) *Defer {
	ninstr := &Defer{
		pos: instr.pos,
	}
	c.remember(instr, ninstr)
	ninstr.anInstruction = c.cloneAnInstruction(instr.anInstruction)
	ninstr.Call = c.cloneCallCommon(instr.Call)
	return ninstr
}

func (c *cloner) cloneSend(instr *Send) *Send {
	ninstr := &Send{
		pos: instr.pos,
	}
	c.remember(instr, ninstr)
	ninstr.Chan = c.cloneValue(instr.Chan)
	ninstr.X = c.cloneValue(instr.X)
	return ninstr
}

func (c *cloner) cloneStore(instr *Store) *Store {
	ninstr := &Store{
		pos: instr.pos,
	}
	c.remember(instr, ninstr)
	ninstr.Addr = c.cloneValue(instr.Addr)
	ninstr.Val = c.cloneValue(instr.Val)
	return ninstr
}

func (c *cloner) cloneBlankStore(instr *BlankStore) *BlankStore {
	ninstr := &BlankStore{}
	c.remember(instr, ninstr)
	ninstr.anInstruction = c.cloneAnInstruction(instr.anInstruction)
	ninstr.Val = c.cloneValue(instr.Val)
	return ninstr
}

func (c *cloner) cloneMapUpdate(instr *MapUpdate) *MapUpdate {
	ninstr := &MapUpdate{
		pos: instr.pos,
	}
	c.remember(instr, ninstr)
	ninstr.Map = c.cloneValue(instr.Map)
	ninstr.Key = c.cloneValue(instr.Key)
	ninstr.Value = c.cloneValue(instr.Value)
	return ninstr
}

func (c *cloner) cloneDebugRef(instr *DebugRef) *DebugRef {
	ninstr := &DebugRef{
		Expr:   instr.Expr,
		object: instr.object,
		IsAddr: instr.IsAddr,
	}
	c.remember(instr, ninstr)
	ninstr.anInstruction = c.cloneAnInstruction(instr.anInstruction)
	ninstr.X = c.cloneValue(instr.X)
	return ninstr
}

func (c *cloner) cloneBasicBlock(bb *BasicBlock) *BasicBlock {
	// called directly
	if bb == nil {
		return nil
	}
	if nbb, ok := c.bbMap[bb]; ok {
		return nbb
	}

	nbb := &BasicBlock{
		Index:     bb.Index,
		Comment:   bb.Comment,
		gaps:      bb.gaps,
		rundefers: bb.rundefers,
	}
	c.bbMap[bb] = nbb
	nbb.parent = c.cloneFunction(bb.parent)
	bb.Instrs = c.cloneInstructions(bb.Instrs)
	if bb.Preds != nil {
		nbb.Preds = make([]*BasicBlock, len(bb.Preds))
		for i, pred := range bb.Preds {
			nbb.Preds[i] = c.cloneBasicBlock(pred)
		}
	}
	if bb.Succs != nil {
		nbb.Succs = make([]*BasicBlock, len(bb.Succs))
		for i, succ := range bb.Succs {
			nbb.Succs[i] = c.cloneBasicBlock(succ)
		}
	}
	for i, succ := range bb.succs2 {
		nbb.succs2[i] = c.cloneBasicBlock(succ)
	}
	nbb.dom.idom = bb.dom.idom
	if bb.dom.children != nil {
		nbb.dom.children = make([]*BasicBlock, len(bb.dom.children))
		for i, child := range bb.dom.children {
			nbb.dom.children[i] = c.cloneBasicBlock(child)
		}
	}
	nbb.dom.pre = bb.dom.pre
	nbb.dom.post = bb.dom.post

	return nbb
}

func (c *cloner) cloneAnInstruction(instr anInstruction) anInstruction {
	return anInstruction{
		block: c.cloneBasicBlock(instr.block),
	}
}

func (c *cloner) cloneRegister(reg register) register {
	nreg := register{
		anInstruction: c.cloneAnInstruction(reg.anInstruction),
		num:           reg.num,
		typ:           reg.typ,
		pos:           reg.pos,
	}
	reg.referrers = c.cloneInstructions(reg.referrers)
	return nreg
}

func (c *cloner) cloneInstructions(instrs []Instruction) []Instruction {
	if instrs == nil {
		return nil
	}
	ninstrs := make([]Instruction, len(instrs))
	for i, instr := range instrs {
		ninstrs[i] = c.cloneInstruction(instr)
	}
	return ninstrs
}

func (c *cloner) cloneValues(values []Value) []Value {
	if values == nil {
		return nil
	}
	nvalues := make([]Value, len(values))
	for i, value := range values {
		nvalues[i] = c.cloneValue(value)
	}
	return nvalues
}

// Clone creates a deep copy of prog. This can be used to retain a
// copy of the naive form before performing lifting.
func (prog *Program) Clone() *Program {
	nprog := &Program{
		Fset:       prog.Fset,
		mode:       prog.mode,
		MethodSets: prog.MethodSets,
		imported:   map[string]*Package{},
		packages:   map[*types.Package]*Package{},
	}

	// TODO methodSets, runtimeTypes, canon, bounds, thunks

	c := &cloner{
		nprog:  nprog,
		pkgMap: map[*Package]*Package{},
		theMap: map[interface{}]interface{}{},
		bbMap:  map[*BasicBlock]*BasicBlock{},
	}

	for path, pkg := range prog.imported {
		nprog.imported[path] = c.clonePkg(pkg)
	}
	for tpkg, pkg := range prog.packages {
		nprog.packages[tpkg] = c.clonePkg(pkg)
	}

	// XXX
	return nprog
}
