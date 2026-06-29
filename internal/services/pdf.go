package services

import (
	"fmt"
	"job-agent-api/internal/dto"
	"strings"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

func GeneratePDF(cv dto.GeneratedCv) ([]byte, error) {

	cfg := config.NewBuilder().
		WithLeftMargin(20).
		WithTopMargin(15).
		WithRightMargin(20).
		WithBottomMargin(15).
		Build()

	m := maroto.New(cfg)

	// ==========================
	// HEADER
	// ==========================

	m.AddAutoRow(
		text.NewCol(12, cv.Nome, props.Text{
			Size:  20,
			Style: fontstyle.Bold,
		}),
	)

	contact := fmt.Sprintf("%s | %s | %s | %s",
		cv.Email,
		cv.Telefone,
		cv.Linkedin,
		cv.Github,
	)

	m.AddAutoRow(
		text.NewCol(12, contact, props.Text{
			Size: 10,
		}),
	)

	// ==========================
	// RESUMO
	// ==========================

	if cv.Resumo != "" {
		m.AddAutoRow(
			text.NewCol(12, "Resumo", props.Text{
				Size:  14,
				Style: fontstyle.Bold,
				Top:   6,
			}),
		)

		m.AddAutoRow(
			text.NewCol(12, cv.Resumo, props.Text{
				Size: 11,
				Top:  2,
			}),
		)
	}

	// ==========================
	// SKILLS
	// ==========================

	if len(cv.Skills) > 0 {
		m.AddAutoRow(
			text.NewCol(12, "Skills", props.Text{
				Size:  14,
				Style: fontstyle.Bold,
				Top:   8,
			}),
		)

		m.AddAutoRow(
			text.NewCol(12, join(cv.Skills), props.Text{
				Size: 11,
				Top:  2,
			}),
		)
	}

	// ==========================
	// EXPERIÊNCIA
	// ==========================

	if len(cv.Experiencias) > 0 {
		m.AddAutoRow(
			text.NewCol(12, "Experiência", props.Text{
				Size:  14,
				Style: fontstyle.Bold,
				Top:   8,
			}),
		)

		for _, exp := range cv.Experiencias {
			m.AddAutoRow(
				text.NewCol(12,
					fmt.Sprintf("%s - %s", exp.Cargo, exp.Empresa),
					props.Text{
						Style: fontstyle.Bold,
						Size:  11,
						Top:   4,
					},
				),
			)

			m.AddAutoRow(
				text.NewCol(12,
					fmt.Sprintf("%s a %s", exp.DataInicio, exp.DataFim),
					props.Text{
						Size: 10,
						Top:  1,
					},
				),
			)

			m.AddAutoRow(
				text.NewCol(12, exp.Descricao, props.Text{
					Size: 11,
					Top:  1,
				}),
			)
		}
	}

	// ==========================
	// EDUCAÇÃO
	// ==========================

	if len(cv.Educacao) > 0 {
		m.AddAutoRow(
			text.NewCol(12, "Educação", props.Text{
				Size:  14,
				Style: fontstyle.Bold,
				Top:   8,
			}),
		)

		for _, edu := range cv.Educacao {
			m.AddAutoRow(
				text.NewCol(12,
					fmt.Sprintf("%s - %s", edu.Curso, edu.Instituicao),
					props.Text{
						Style: fontstyle.Bold,
						Size:  11,
						Top:   4,
					},
				),
			)

			m.AddAutoRow(
				text.NewCol(12,
					fmt.Sprintf("%s a %s", edu.DataInicio, edu.DataFim),
					props.Text{
						Size: 10,
						Top:  1,
					},
				),
			)
		}
	}

	// ==========================
	// FOOTER
	// ==========================

	// m.RegisterFooter(
	// 	m.AddAutoRow(
	// 		text.NewCol(12,
	// 			fmt.Sprintf("Gerado em %s UTC", time.Now().UTC().Format("02/01/2006 15:04")),
	// 			props.Text{
	// 				Size:  9,
	// 				Align: align.Center,
	// 			},
	// 		),
	// 	),
	// )

	document, err := m.Generate()
	if err != nil {
		return nil, err
	}

	return document.GetBytes(), nil
}

func join(s []string) string {
	return strings.Join(s, ", ")
}