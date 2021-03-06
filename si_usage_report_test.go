package main_test

import (
	"code.cloudfoundry.org/cli/plugin"
	"code.cloudfoundry.org/cli/plugin/pluginfakes"
	"encoding/json"
	"fmt"
	. "github.com/jpatel-pivotal/si-usage-report"
	"github.com/jpatel-pivotal/si-usage-report/cfapihelper"
	"github.com/jpatel-pivotal/si-usage-report/cfapihelper/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"io"
	"os/exec"
)

var _ bool = Describe("SiUsageReport", func() {
	Describe("happy path", func() {
		var (
			subject                 *SIUsageReport
			expectedPluginVersion   plugin.VersionType
			expectedCLIVersion      plugin.VersionType
			expectedCommand         plugin.Command
			outBuffer               io.Writer
			errBuffer               io.Writer
			fakeCLIConnection       *pluginfakes.FakeCliConnection
			apiHelper               cfapihelper.CFAPIHelper
			fakeapiHelper           fakes.FakeAPIHelper
			expectedServiceInstance []cfapihelper.ServiceInstance_Details
			expectedReport          Report
		)

		BeforeEach(func() {
			subject = new(SIUsageReport)
			fakeCLIConnection = &pluginfakes.FakeCliConnection{}
			apiHelper = cfapihelper.New(fakeCLIConnection)
			subject.CliConnection = fakeCLIConnection
			subject.APIHelper = apiHelper
			expectedPluginVersion = plugin.VersionType{
				Major: 1,
				Minor: 0,
				Build: 0,
			}
			expectedCLIVersion = plugin.VersionType{
				Major: 6,
				Minor: 7,
				Build: 0,
			}
			expectedCommand = plugin.Command{
				Name:     "si-usage-report",
				HelpText: "Shows Service Instance Usage Report",
				UsageDetails: plugin.Usage{
					Usage: "si-usage-report\n   cf si-usage-report",
				},
			}
			outBuffer = gbytes.NewBuffer()
			subject.OutBuf = outBuffer
			errBuffer = gbytes.NewBuffer()
		})

		Context("GetMetaData() is called", func() {
			It("returns the correct name for the plugin", func() {
				Expect(subject.GetMetadata().Name).To(Equal("si-usage-report"))
			})
			It("returns the correct version of the plugin", func() {
				Expect(subject.GetMetadata().Version).To(Equal(expectedPluginVersion))
			})
			It("returns the correct min CLI version of the plugin", func() {
				Expect(subject.GetMetadata().MinCliVersion).To(Equal(expectedCLIVersion))
			})
			It("returns the correct command", func() {
				Expect(len(subject.GetMetadata().Commands)).To(Equal(1))
				Expect(subject.GetMetadata().Commands[0]).To(Equal(expectedCommand))
			})
		})
		//TODO Move this to integration test
		Context("cf si-usage-report is run without installing the plugin", func() {
			BeforeEach(func() {
				args :=[]string{"uninstall-plugin", "si-usage-report"}
				session, err := gexec.Start(exec.Command("cf", args...), outBuffer, errBuffer)
				session.Wait()
				Expect(err).NotTo(HaveOccurred())
			})
			subject = new(SIUsageReport)
			fakeCLIConnection = &pluginfakes.FakeCliConnection{}
			apiHelper = cfapihelper.New(fakeCLIConnection)
			subject.CliConnection = fakeCLIConnection
			subject.APIHelper = apiHelper
			outBuffer = gbytes.NewBuffer()
			subject.OutBuf = outBuffer
			subject.Run(fakeCLIConnection, []string{"si-usage-report"})
			//subject.GetSIUsageReport([]string{"test"})
			It("prints generic CF CLI message", func() {
				args := []string{"si-usage-report"}
				session, err := gexec.Start(exec.Command("cf", args...), outBuffer, errBuffer)
				session.Wait()
				Expect(err).NotTo(HaveOccurred())
				Expect(outBuffer).To(gbytes.Say("'si-usage-report' is not a registered command. See 'cf help -a'"))
				Expect(errBuffer).To(gbytes.Say(""))
			})
		})
		Context("GetSIUsageReport is called", func() {
			When("user is not logged in", func() {
				BeforeEach(func() {
					fakeCLIConnection.IsLoggedInReturns(false, nil)
					subject.CliConnection = fakeCLIConnection

				})
				It("asks user to log in", func() {
					subject.GetSIUsageReport([]string{"test"})
					Expect(outBuffer).To(gbytes.Say("error: not logged in.\n run cf login"))
				})
			})
			When("user is logged in", func() {
				When("service instances are not returned", func() {
					BeforeEach(func() {
						fakeCLIConnection.IsLoggedInReturns(true, nil)
						subject.CliConnection = fakeCLIConnection

					})
					It("prints an error message", func() {
						subject.GetSIUsageReport([]string{"test"})
						Expect(outBuffer).To(gbytes.Say("error while getting service instances: CF API returned no output"))
					})
				})
				When("service instances returned are not p.redis, p.pcc, p.mysql or p.rabbit", func() {
					var expectedSIJSON []byte
					var err error
					BeforeEach(func() {
						fakeCLIConnection.IsLoggedInReturns(true, nil)
						fakeapiHelper = fakes.FakeAPIHelper{
							CliConnection: fakeCLIConnection,
						}
						subject.APIHelper = &fakeapiHelper
						subject.CliConnection = fakeCLIConnection
						expectedServiceInstance = []cfapihelper.ServiceInstance_Details{}
						expectedSIJSON, err = json.Marshal(expectedServiceInstance)
						Expect(err).NotTo(HaveOccurred())
						Expect(expectedSIJSON).ToNot(BeNil())
					})
					It("returns an empty list", func() {
						report := subject.GenerateReport(expectedServiceInstance)
						fmt.Println(report)
						Expect(len(report.Products)).To(Equal(0))
					})
				})
				When("service instances returned are p.redis, p.pcc, p.mysql or p.rabbit", func() {
					var expectedSIJSON []byte
					var err error
					BeforeEach(func() {
						fakeCLIConnection.IsLoggedInReturns(true, nil)
						fakeapiHelper = fakes.FakeAPIHelper{
							CliConnection: fakeCLIConnection,
						}
						subject.APIHelper = &fakeapiHelper
						subject.CliConnection = fakeCLIConnection
						expectedReport = Report{
							Products: []Product{
								{
									Name: "p.mysql",
									Plans: []Plan{
										{
											PlanName:      "10mb",
											InstanceCount: 3,
										},
										{
											PlanName:      "100mb",
											InstanceCount: 1,
										},
									},
								},
								{
									Name: "p.pcc",
									Plans: []Plan{
										{
											PlanName:      "small",
											InstanceCount: 1,
										},
									},
								},
								{
									Name: "p.rabbit",
									Plans: []Plan{
										{
											PlanName:      "lemur",
											InstanceCount: 2,
										},
									},
								},
								{
									Name: "p.redis",
									Plans: []Plan{
										{
											PlanName:      "medium",
											InstanceCount: 1,
										},
									},
								},
							},
						}
						expectedSIJSON, err = json.Marshal(expectedReport)
						Expect(err).NotTo(HaveOccurred())
						Expect(expectedSIJSON).ToNot(BeNil())
					})
					It("prints service instance list in json", func() {
						subject.GetSIUsageReport([]string{"test"})
						outJSON := outBuffer.(*gbytes.Buffer).Contents()
						Expect(outJSON).Should(MatchJSON(expectedSIJSON))
					})

				})
				When("a lot of service instances of type p.redis, p.pcc, p.mysql or p.rabbit are returned", func() {
					var expectedSIJSON []byte
					var serviceInstancesJSON []string
					var err error
					BeforeEach(func() {
						fakeCLIConnection.IsLoggedInReturns(true, nil)
						serviceInstancesJSON = fakeapiHelper.GetResponse("cfapihelper/test-data/lot-of-service-instances.json")
						fakeCLIConnection.CliCommandWithoutTerminalOutputReturns(serviceInstancesJSON, nil)
						newAPIHelper := cfapihelper.New(fakeCLIConnection)
						subject.APIHelper = newAPIHelper
						subject.CliConnection = fakeCLIConnection
						expectedReport = Report{
							Products: []Product{
								{
									Name: "p.mysql",
									Plans: []Plan{
										{
											PlanName:      "panda",
											InstanceCount: 1654,
										},
										{
											PlanName:      "turtle",
											InstanceCount: 6616,
										},
										{
											PlanName:      "hippo",
											InstanceCount: 827,
										},
									},
								},
							},
						}
						expectedSIJSON, err = json.Marshal(expectedReport)
						Expect(err).NotTo(HaveOccurred())
						Expect(expectedSIJSON).ToNot(BeNil())
					})
					It("prints service instance list in json", func() {
						subject.GetSIUsageReport([]string{"test"})
						outJSON := outBuffer.(*gbytes.Buffer).Contents()
						Expect(outJSON).Should(MatchJSON(expectedSIJSON))
					})
					Measure("it should do this efficiently", func(b Benchmarker) {
						runtime := b.Time("runtime", func() {
							subject.GetSIUsageReport([]string{"test"})

						})

						Ω(runtime.Seconds()).Should(BeNumerically("<", 7), "GetSIUsageReport() shouldn't take too long.")

						//b.RecordValue("disk usage (in MB)", HowMuchDiskSpaceDidYouUse())
					}, 10)
				})
			})
		})
	})

})
