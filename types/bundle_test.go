package types

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Bundle", func() {
	Describe("IsPaid", func() {
		It("should return true when bundle has PaidSkus", func() {
			bundle := Bundle{
				Name:     "test-bundle",
				PaidSkus: []string{"SKU1", "SKU2"},
			}
			Expect(bundle.IsPaid()).To(BeTrue())
		})

		It("should return true when bundle has EvalSkus", func() {
			bundle := Bundle{
				Name:     "test-bundle",
				EvalSkus: []string{"EVAL1", "EVAL2"},
			}
			Expect(bundle.IsPaid()).To(BeTrue())
		})

		It("should return true when bundle has both PaidSkus and EvalSkus", func() {
			bundle := Bundle{
				Name:     "test-bundle",
				PaidSkus: []string{"SKU1"},
				EvalSkus: []string{"EVAL1"},
			}
			Expect(bundle.IsPaid()).To(BeTrue())
		})

		It("should return false when bundle has only regular Skus", func() {
			bundle := Bundle{
				Name: "test-bundle",
				Skus: []string{"SKU1", "SKU2"},
			}
			Expect(bundle.IsPaid()).To(BeFalse())
		})

		It("should return false when bundle has empty PaidSkus and EvalSkus", func() {
			bundle := Bundle{
				Name:     "test-bundle",
				PaidSkus: []string{},
				EvalSkus: []string{},
			}
			Expect(bundle.IsPaid()).To(BeFalse())
		})

		It("should return false when bundle has nil PaidSkus and EvalSkus", func() {
			bundle := Bundle{
				Name: "test-bundle",
			}
			Expect(bundle.IsPaid()).To(BeFalse())
		})

		It("should return true when bundle has empty PaidSkus but non-empty EvalSkus", func() {
			bundle := Bundle{
				Name:     "test-bundle",
				PaidSkus: []string{},
				EvalSkus: []string{"EVAL1"},
			}
			Expect(bundle.IsPaid()).To(BeTrue())
		})
	})

	Describe("IsSkuBased", func() {
		It("should return true when bundle has regular Skus", func() {
			bundle := Bundle{
				Name: "test-bundle",
				Skus: []string{"SKU1", "SKU2"},
			}
			Expect(bundle.IsSkuBased()).To(BeTrue())
		})

		It("should return true when bundle has PaidSkus", func() {
			bundle := Bundle{
				Name:     "test-bundle",
				PaidSkus: []string{"PAID1", "PAID2"},
			}
			Expect(bundle.IsSkuBased()).To(BeTrue())
		})

		It("should return true when bundle has EvalSkus", func() {
			bundle := Bundle{
				Name:     "test-bundle",
				EvalSkus: []string{"EVAL1", "EVAL2"},
			}
			Expect(bundle.IsSkuBased()).To(BeTrue())
		})

		It("should return true when bundle has regular Skus and PaidSkus", func() {
			bundle := Bundle{
				Name:     "test-bundle",
				Skus:     []string{"SKU1"},
				PaidSkus: []string{"PAID1"},
			}
			Expect(bundle.IsSkuBased()).To(BeTrue())
		})

		It("should return false when bundle has no SKUs at all", func() {
			bundle := Bundle{
				Name:           "test-bundle",
				UseValidAccNum: true,
			}
			Expect(bundle.IsSkuBased()).To(BeFalse())
		})

		It("should return false when bundle has empty Skus arrays", func() {
			bundle := Bundle{
				Name:     "test-bundle",
				Skus:     []string{},
				PaidSkus: []string{},
				EvalSkus: []string{},
			}
			Expect(bundle.IsSkuBased()).To(BeFalse())
		})

		It("should return false when bundle has nil Skus", func() {
			bundle := Bundle{
				Name: "test-bundle",
			}
			Expect(bundle.IsSkuBased()).To(BeFalse())
		})
	})
})
