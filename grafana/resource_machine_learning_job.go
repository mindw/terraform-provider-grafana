package grafana

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/grafana/machine-learning-go-client/mlapi"
)

func ResourceMachineLearningJob() *schema.Resource {
	return &schema.Resource{

		Description: `
A job defines the queries and model parameters for a machine learning task.
`,

		CreateContext: resourceMachineLearningJobCreate,
		ReadContext:   resourceMachineLearningJobRead,
		UpdateContext: resourceMachineLearningJobUpdate,
		DeleteContext: resourceMachineLearningJobDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"id": {
				Description: "The ID of the job.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"name": {
				Description: "The name of the job.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"metric": {
				Description: "The metric used to query the job results.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"description": {
				Description: "A description of the job.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"datasource_id": {
				Description: "The id of the datasource to query.",
				Type:        schema.TypeInt,
				Optional:    true,
			},
			"datasource_uid": {
				Description: "The uid of the datasource to query.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"datasource_type": {
				Description: "The type of datasource being queried. Currently allowed values are prometheus, graphite, loki, postgres, and datadog.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"query_params": {
				Description: "An object representing the query params to query Grafana with.",
				Type:        schema.TypeMap,
				Required:    true,
			},
			"interval": {
				Description: "The data interval in seconds to train the data on.",
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     300,
			},
			"hyper_params": {
				Description: "The hyperparameters used to fine tune the algorithm. See https://grafana.com/docs/grafana-cloud/machine-learning/models/ for the full list of available hyperparameters.",
				Type:        schema.TypeMap,
				Optional:    true,
				Default:     map[string]interface{}{},
			},
			"training_window": {
				Description: "The data interval in seconds to train the data on.",
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     int(90 * 24 * time.Hour / time.Second),
			},
		},
	}
}

func resourceMachineLearningJobCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client).mlapi
	job, err := makeMLJob(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	job, err = c.NewJob(ctx, job)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(job.ID)
	return resourceMachineLearningJobRead(ctx, d, meta)
}

func resourceMachineLearningJobRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client).mlapi
	job, err := c.Job(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("name", job.Name)
	d.Set("metric", job.Metric)
	d.Set("description", job.Description)
	if job.DatasourceID != 0 {
		d.Set("datasource_id", job.DatasourceID)
	} else {
		d.Set("datasource_id", nil)
	}
	if job.DatasourceUID != "" {
		d.Set("datasource_uid", job.DatasourceUID)
	} else {
		d.Set("datasource_uid", nil)
	}
	d.Set("datasource_type", job.DatasourceType)
	d.Set("query_params", job.QueryParams)
	d.Set("interval", job.Interval)
	d.Set("hyper_params", job.HyperParams)
	d.Set("training_window", job.TrainingWindow)

	return nil
}

func resourceMachineLearningJobUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client).mlapi
	job, err := makeMLJob(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	_, err = c.UpdateJob(ctx, job)
	if err != nil {
		return diag.FromErr(err)
	}
	return resourceMachineLearningJobRead(ctx, d, meta)
}

func resourceMachineLearningJobDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client).mlapi
	err := c.DeleteJob(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

func makeMLJob(d *schema.ResourceData, meta interface{}) (mlapi.Job, error) {
	datasourceID := uint(d.Get("datasource_id").(int))
	datasourceUID := d.Get("datasource_uid").(string)
	if datasourceID == 0 && datasourceUID == "" {
		return mlapi.Job{}, fmt.Errorf("either datasource_id or datasource_uid must be set")
	}
	return mlapi.Job{
		ID:                d.Id(),
		Name:              d.Get("name").(string),
		Metric:            d.Get("metric").(string),
		Description:       d.Get("description").(string),
		GrafanaURL:        meta.(*client).gapiURL,
		DatasourceID:      datasourceID,
		DatasourceUID:     datasourceUID,
		DatasourceType:    d.Get("datasource_type").(string),
		QueryParams:       d.Get("query_params").(map[string]interface{}),
		Interval:          uint(d.Get("interval").(int)),
		Algorithm:         "Prophet",
		HyperParams:       d.Get("hyper_params").(map[string]interface{}),
		TrainingWindow:    uint(d.Get("training_window").(int)),
		TrainingFrequency: uint(24 * time.Hour / time.Second),
	}, nil
}
