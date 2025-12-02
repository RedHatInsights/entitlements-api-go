package main

import (
	"os"

	"github.com/RedHatInsights/entitlements-api-go/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
)

var _ = Describe("Bundle Sync", func() {
	Describe("uniqueSkus", func() {
		It("should remove duplicate SKUs", func() {
			input := []string{"SKU1", "SKU2", "SKU1", "SKU3", "SKU2"}
			result := uniqueSkus(input)

			Expect(len(result)).To(Equal(3))
			Expect(result).To(ContainElement("SKU1"))
			Expect(result).To(ContainElement("SKU2"))
			Expect(result).To(ContainElement("SKU3"))
		})

		It("should handle empty slice", func() {
			input := []string{}
			result := uniqueSkus(input)

			Expect(len(result)).To(Equal(0))
		})

		It("should handle slice with no duplicates", func() {
			input := []string{"SKU1", "SKU2", "SKU3"}
			result := uniqueSkus(input)

			Expect(len(result)).To(Equal(3))
			Expect(result).To(ContainElement("SKU1"))
			Expect(result).To(ContainElement("SKU2"))
			Expect(result).To(ContainElement("SKU3"))
		})

		It("should handle slice with all duplicates", func() {
			input := []string{"SKU1", "SKU1", "SKU1"}
			result := uniqueSkus(input)

			Expect(len(result)).To(Equal(1))
			Expect(result).To(ContainElement("SKU1"))
		})

		It("should return sorted results", func() {
			input := []string{"SKU3", "SKU1", "SKU4", "SKU2", "SKU3"}
			result := uniqueSkus(input)

			Expect(len(result)).To(Equal(4))
			Expect(result).To(Equal([]string{"SKU1", "SKU2", "SKU3", "SKU4"}))
		})
	})

	Describe("getBundlesConfig", func() {
		var cfg *viper.Viper

		BeforeEach(func() {
			cfg = viper.New()
			cfg.Set(config.Keys.PaidFeatureSuffix, "_paid")
		})

		It("should load bundles from YAML file", func() {
			cfg.Set(config.Keys.BundleInfoYaml, "../test_data/test_bundle.yml")

			bundlesMap, err := getBundlesConfig(cfg)

			Expect(err).To(BeNil())
			Expect(bundlesMap).ToNot(BeEmpty())
			Expect(bundlesMap).To(HaveKey("TestBundle1"))
		})

		It("should return error when file does not exist", func() {
			cfg.Set(config.Keys.BundleInfoYaml, "nonexistent_file.yml")

			bundlesMap, err := getBundlesConfig(cfg)

			Expect(err).ToNot(BeNil())
			Expect(bundlesMap).To(BeEmpty())
		})

		Context("When bundle has paid and eval SKUs", func() {
			var testYamlPath string

			BeforeEach(func() {
				// Create a temporary test file with paid/eval SKUs
				testYaml := `- name: PaidBundle
  paid_skus:
    - PAID1
    - PAID2
  eval_skus:
    - EVAL1
    - EVAL2

- name: RegularBundle
  skus:
    - SKU1
    - SKU2
`
				tmpFile, err := os.CreateTemp("", "test_bundle_*.yml")
				Expect(err).To(BeNil())
				testYamlPath = tmpFile.Name()

				_, err = tmpFile.Write([]byte(testYaml))
				Expect(err).To(BeNil())
				tmpFile.Close()

				cfg.Set(config.Keys.BundleInfoYaml, testYamlPath)
			})

			AfterEach(func() {
				os.Remove(testYamlPath)
			})

			It("should create both base and _paid bundles for paid features", func() {
				bundlesMap, err := getBundlesConfig(cfg)

				Expect(err).To(BeNil())
				Expect(bundlesMap).To(HaveKey("PaidBundle"))
				Expect(bundlesMap).To(HaveKey("PaidBundle_paid"))

				// Base bundle should have all SKUs (eval + paid)
				baseBundle := bundlesMap["PaidBundle"]
				Expect(baseBundle.Skus).To(ContainElement("PAID1"))
				Expect(baseBundle.Skus).To(ContainElement("PAID2"))
				Expect(baseBundle.Skus).To(ContainElement("EVAL1"))
				Expect(baseBundle.Skus).To(ContainElement("EVAL2"))

				// Paid bundle should only have paid SKUs
				paidBundle := bundlesMap["PaidBundle_paid"]
				Expect(paidBundle.Skus).To(ContainElement("PAID1"))
				Expect(paidBundle.Skus).To(ContainElement("PAID2"))
				Expect(len(paidBundle.Skus)).To(Equal(2))
			})

			It("should not create _paid bundle for regular bundles", func() {
				bundlesMap, err := getBundlesConfig(cfg)

				Expect(err).To(BeNil())
				Expect(bundlesMap).To(HaveKey("RegularBundle"))
				Expect(bundlesMap).ToNot(HaveKey("RegularBundle_paid"))
			})

			It("should remove duplicate SKUs when creating bundles", func() {
				// Create test file with duplicate SKUs
				testYaml := `- name: DuplicateBundle
  paid_skus:
    - SKU1
    - SKU2
  eval_skus:
    - SKU1
    - SKU3
`
				tmpFile, err := os.CreateTemp("", "test_dup_*.yml")
				Expect(err).To(BeNil())
				dupYamlPath := tmpFile.Name()

				_, err = tmpFile.Write([]byte(testYaml))
				Expect(err).To(BeNil())
				tmpFile.Close()
				defer os.Remove(dupYamlPath)

				cfg.Set(config.Keys.BundleInfoYaml, dupYamlPath)

				bundlesMap, err := getBundlesConfig(cfg)

				Expect(err).To(BeNil())
				baseBundle := bundlesMap["DuplicateBundle"]

				// Should have 3 unique SKUs (SKU1, SKU2, SKU3)
				Expect(len(baseBundle.Skus)).To(Equal(3))
				Expect(baseBundle.Skus).To(ContainElement("SKU1"))
				Expect(baseBundle.Skus).To(ContainElement("SKU2"))
				Expect(baseBundle.Skus).To(ContainElement("SKU3"))
			})

			It("should preserve bundle attributes in both base and paid bundles", func() {
				testYaml := `- name: AttributeBundle
  use_valid_acc_num: true
  use_valid_org_id: true
  use_is_internal: false
  paid_skus:
    - PAID1
  eval_skus:
    - EVAL1
`
				tmpFile, err := os.CreateTemp("", "test_attr_*.yml")
				Expect(err).To(BeNil())
				attrYamlPath := tmpFile.Name()

				_, err = tmpFile.Write([]byte(testYaml))
				Expect(err).To(BeNil())
				tmpFile.Close()
				defer os.Remove(attrYamlPath)

				cfg.Set(config.Keys.BundleInfoYaml, attrYamlPath)

				bundlesMap, err := getBundlesConfig(cfg)

				Expect(err).To(BeNil())

				baseBundle := bundlesMap["AttributeBundle"]
				Expect(baseBundle.UseValidAccNum).To(BeTrue())
				Expect(baseBundle.UseValidOrgId).To(BeTrue())
				Expect(baseBundle.UseIsInternal).To(BeFalse())

				paidBundle := bundlesMap["AttributeBundle_paid"]
				Expect(paidBundle.UseValidAccNum).To(BeTrue())
				Expect(paidBundle.UseValidOrgId).To(BeTrue())
				Expect(paidBundle.UseIsInternal).To(BeFalse())
			})
		})
	})
})
