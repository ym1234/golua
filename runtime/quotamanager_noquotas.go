//go:build noquotas
// +build noquotas

package runtime

const QuotasAvailable = false

type quotaManager struct{}

var _ RuntimeContext = (*quotaManager)(nil)

func (m *quotaManager) CpuLimit() uint64 {
	return 0
}

func (m *quotaManager) CpuUsed() uint64 {
	return 0
}

func (m *quotaManager) MemLimit() uint64 {
	return 0
}

func (m *quotaManager) MemUsed() uint64 {
	return 0
}

func (m *quotaManager) Status() RuntimeContextStatus {
	return RCS_Live
}

func (m *quotaManager) Parent() RuntimeContext {
	return m
}

func (m *quotaManager) RuntimeContext() RuntimeContext {
	return m
}

func (m *quotaManager) PushContext(ctx RuntimeContext) {
}

func (m *quotaManager) PopContext() RuntimeContext {
	return m
}

func (m *quotaManager) PushQuota(cpuQuota, memQuota uint64) {
}

func (m *quotaManager) PopQuota() {
}

func (m *quotaManager) AllowQuotaModificationsInLua() {
}

func (m *quotaManager) QuotaModificationsInLuaAllowed() bool {
	return false
}

func (m *quotaManager) RequireCPU(cpuAmount uint64) {
}

func (m *quotaManager) UpdateCPUQuota(newQuota uint64) {
}

func (m *quotaManager) UnusedCPU() uint64 {
	return 0
}

func (m *quotaManager) CPUQuotaStatus() (uint64, uint64) {
	return 0, 0
}

func (m *quotaManager) RequireMem(memAmount uint64) {
}

func (m *quotaManager) RequireSize(sz uintptr) uint64 {
	return 0
}

func (m *quotaManager) RequireArrSize(sz uintptr, n int) uint64 {
	return 0
}

func (m *quotaManager) RequireBytes(n int) uint64 {
	return 0
}

func (m *quotaManager) ReleaseMem(memAmount uint64) {
}

func (m *quotaManager) ReleaseSize(sz uintptr) {
}

func (m *quotaManager) ReleaseArrSize(sz uintptr, n int) {
}

func (m *quotaManager) ReleaseBytes(n int) {
}

func (m *quotaManager) UpdateMemQuota(newQuota uint64) {
}

func (m *quotaManager) UnusedMem() uint64 {
	return 0
}

func (m *quotaManager) MemQuotaStatus() (uint64, uint64) {
	return 0, 0
}

func (m *quotaManager) LinearUnused(cpuFactor uint64) uint64 {
	return 0
}

func (m *quotaManager) LinearRequire(cpuFactor uint64, amt uint64) {
}

func (m *quotaManager) ResetQuota() {
}