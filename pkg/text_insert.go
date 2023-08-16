package thd

import (
	"database/sql"
	"fmt"
	"strings"
	"unicode"

	_ "github.com/lib/pq"
)

func sub(contents string) []string {
	f := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	}
	arr := strings.FieldsFunc(contents, f)
	return arr
}

func Insert(host, port, user, password, dbname, pcName, joborder, contents string) (string, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		fmt.Sprintln("failed to open database connection: %w", err)
	}
	defer db.Close()

	sqlStatement := `
INSERT INTO coding_data (
    chip_tag, reserved, family_id, approved_hp_oem, oem_id, address_position,template_version_major, template_version_minor, tag_encryption_mask, person_revision, reserved_for_perso, 
	ud0_fuse, dry_cartridge_sn_manufacture_site_id, dry_cartridge_sn_manufacture_line, dry_cartridge_sn_manufacture_year, dry_cartridge_sn_week_of_year, dry_cartridge_sn_day_of_week, dry_cartridge_sn_hour_of_day, dry_cartridge_sn_minute_of_hour, dry_cartridge_sn_sec_of_minute, dry_cartridge_sn_process_position, max_usable_cartridge_volume, printer_lockdown_parition, 
	thermal_sense_resistor_tsr, tsr_thermal_coeffcient_tcr, bulk, 
	ud1_fuse, cartridge_fill_sn_site_id, cartridge_fill_sn_line, cartridge_fill_sn_year, cartridge_fill_sn_week_of_year, cartridge_fill_sn_day_of_week, cartridge_fill_sn_hour_of_day, cartridge_fill_sn_minute_of_hour, cartridge_fill_sn_sec_of_minute, cartridge_fill_sn_process_position,
	ink_formulator_id, ink_family, color_codes_general, color_codes_specific, ink_family_member, ink_id_number, ink_revision, ink_density, cartridge_distinction, supply_key_size_descriptor, shelf_life_weeks, shelf_life_days, installed_life_weeks, installed_life_days, usable_ink_weight, altered_supply_notification_level, 
	firing_frequency, pulse_width_tpw, firing_voltage, turn_on_energy_toe, pulse_warming_temperature, maximum_temperature, drop_volume, 
	write_protect_fuse, _1st_platform_id, _1st_platform_manf_year, _1st_platform_manf_week_of_year, _1st_platform_mfg_country, _1st_platform_fw_revision_major, _1st_platform_fw_revision_minor, _1st_install_cartridge_count, cartridge_1st_install_year, cartridge_1st_install_week_of_year, cartridge_1st_install_day_of_week, ink_level_gauge_resolution,
	ud3_fuse, oem_defined_field_1, oem_defined_field_2, 
	trademark_string, ud4_fuse,
	out_of_ink_bit, ilg_bits_1_25, ilg_bits_26_50, ilg_bits_51_75, ilg_bits_76_100, tiug_bits_1_25, tiug_bits_26_50, tiug_bits_51_75, tiug_bits_76_100, first_failure_code, altered_supply, user_acknowledge_altered_supply, user_acknowledge_expired_ink, faulty_replace_imeediately, 
	oem_defined_rw_or_field_1, oem_defined_rw_or_field_2, 
	cartridge_mru_year, cartridge_mru_week_of_year, cartridge_mru_day_of_week, mru_platform_mfg_year, mru_platform_mfg_week_of_year, mru_platform_mfg_country, mru_platform_fw_revision_major, mru_platform_fw_revision_minor, cartridge_insertion_count, stall_insertion_count, last_failure_code, last_user_reported_status, marketing_data_revision,
    oem_defined_rw_field_1, oem_defined_rw_field_2,
	ud7_fuse, extended_oem_id, hp_oem_ink_designator,
    regionalization_id, cartridge_reorder_pn, ud8_fuse, pcname, joborder
)
VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
	$21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37, $38,
	$39, $40, $41, $42, $43, $44, $45, $46, $47, $48, $49, $50, $51, $52, $53, $54, $55, $56,
	$57, $58, $59, $60, $61, $62, $63, $64, $65, $66, $67, $68, $69, $70, $71, $72, $73, $74,
	$75, $76, $77, $78, $79, $80, $81, $82, $83, $84, $85, $86, $87, $88, $89, $90, $91, $92,
	$93, $94, $95, $96, $97, $98, $99, $100, $101, $102, $103, $104, $105, $106, $107, $108,
	$109, $110, $111, $112, $113, $114, $115
)
RETURNING uuid
`

	var uuid string
	Slice := sub(contents)
	if len(Slice) == 0 {
		Slice = []string{"0000"}
	}

	arguments := make([]interface{}, 0, 114)
	for _, val := range Slice {

		arguments = append(arguments, val)
		//fmt.Printf("Slice %d: %v\n", i+1, val)
	}

	totalToRemove := len(arguments) - 113
	indexToRemove := 113 // Subtract 1 since slices are zero-indexed
	arguments = append(arguments[:indexToRemove], arguments[indexToRemove+totalToRemove:]...)
	arguments = append(arguments, pcName, joborder)

	err = db.QueryRow(sqlStatement, arguments...).Scan(&uuid)
	if err != nil {
		return "failed", fmt.Errorf("failed to execute SQL query: %w", err)
	}

	fmt.Println("Inserted row", uuid)

	return "complete", nil
}
